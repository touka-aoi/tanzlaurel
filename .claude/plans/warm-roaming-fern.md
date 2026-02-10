# 異常切断時にセッションがRoomに残るバグの修正

## Context

ブラウザリロード等でWebSocket接続が正常なLeaveなしに切断されると、セッションが`Room.sessions`に永久に残り続けるバグがある。

原因は2つ:
1. `evReadError`/`evWriteError`が`handleControlEvent`で無視されており、`close()`が呼ばれない
2. `close()`がRoomからのセッション離脱処理を行っていない

ユーザーの方針:
- `evReadError`/`evWriteError`/`evDispatchError`を廃止し、エラー発生箇所から直接`evClose`を送信してclose処理を統一する
- `close()`内でRoomからの離脱処理を追加する

## 変更内容

### 0. ADR-008 の修正 & NewSessionEndpoint の context 受け取り

ADR-008のConsiderationを簡潔化する。`context.WithCancel`は親contextのキャンセルを子に伝播するため、
`context.WithCancel(r.Context())`とするだけでサーバーShutdown時の伝播が成立する。

**`server/domain/session_endpoint.go` — NewSessionEndpoint**:
引数に`ctx context.Context`を追加し、`context.Background()`を`ctx`に置き換える。

```go
func NewSessionEndpoint(ctx context.Context, session *Session, connection *Connection, pubsub PubSub, roomManager RoomManager) (*SessionEndpoint, error) {
    // ...
    ctx, cancel := context.WithCancel(ctx)
    // ...
}
```

**`server/handler/accept.go` — ServeHTTP**:
`r.Context()`を渡す。

```go
endpoint, err := domain.NewSessionEndpoint(r.Context(), session, connection, h.pubsub, h.roomManager)
```

### 1. `endpoint_events.go` — 不要なイベント種別の削除

`evReadError`, `evWriteError`, `evDispatchError` を削除する。

```go
const (
    unknown endpointEventKind = iota
    evPong
    evClose
)
```

### 2. `session_endpoint.go` — readLoop/writeLoopの変更

**readLoop**: `evReadError` → `evClose`に変更し、送信後`return`で即座にループを抜ける

```go
func (se *SessionEndpoint) readLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            data, err := se.connection.Read(ctx)
            if err != nil {
                se.sendCtrlEvent(ctx, endpointEvent{kind: evClose, err: err})
                return  // 接続死亡なのでループ終了
            }
            se.handleData(ctx, data)
        }
    }
}
```

**writeLoop**: `evWriteError` → `evClose`に変更し、送信後`return`

```go
func (se *SessionEndpoint) writeLoop(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case data := <-se.writeCh:
            err := se.connection.Write(ctx, data)
            if err != nil {
                se.sendCtrlEvent(ctx, endpointEvent{kind: evClose, err: err})
                return  // 接続死亡なのでループ終了
            }
            se.session.TouchWrite()
        }
    }
}
```

### 3. `session_endpoint.go` — handleControlEventの簡素化

`evReadError`/`evWriteError`/`evDispatchError`のcaseを削除。

```go
func (se *SessionEndpoint) handleControlEvent(ctx context.Context, ev endpointEvent) {
    switch ev.kind {
    case evClose:
        se.close()
    case evPong:
        se.session.TouchPong()
    default:
        slog.WarnContext(ctx, "unknown endpoint event kind", "kind", ev.kind)
    }
}
```

### 4. `protocol.go` — EncodeLeaveMessage の追加

`EncodeAssignMessage`と同じパターンでLeaveメッセージのエンコード関数を追加。

```go
func EncodeLeaveMessage(sessionID SessionID) []byte {
    header := Header{
        Version:   1,
        SessionID: sessionID.Bytes(),
        Seq:       0,
        Length:    PayloadHeaderSize,
        Timestamp: uint32(time.Now().UnixMilli() & 0xFFFFFFFF),
    }
    payloadHeader := PayloadHeader{
        DataType: DataTypeControl,
        SubType:  uint8(ControlSubTypeLeave),
    }
    data := make([]byte, HeaderSize+PayloadHeaderSize)
    copy(data[:HeaderSize], header.Encode())
    copy(data[HeaderSize:], payloadHeader.Encode())
    return data
}
```

### 5. `session_endpoint.go` — close()にRoom離脱処理を追加

`se.cancel()`の**前に**、Roomに参加している場合はLeaveメッセージをpublishする。

```go
func (se *SessionEndpoint) close() {
    if !se.closed.CompareAndSwap(false, true) {
        return
    }
    // Roomからの離脱（cancel前に実行）
    if !se.roomID.IsEmpty() {
        roomTopic := Topic("room:" + se.roomID.String())
        leaveMsg := EncodeLeaveMessage(se.session.ID())
        se.pubsub.Publish(se.ctx, roomTopic, Message{
            SessionID: se.session.ID(),
            Data:      leaveMsg,
        })
        se.roomID = RoomID{}
    }
    se.cancel()
    se.session.Close()
    se.connection.Close()
}
```

前提: `se.ctx`がキャンセルされるのはサーバー終了時のみ。`evClose`送信時点ではcontextは生きているため、`se.ctx`をそのまま使用できる。

### 6. `session_endpoint.go` — ForceClose()の削除

`ForceClose()`メソッドを削除する。`close()`の呼び出し元はownerLoop内の`handleControlEvent`に統一。

## 対象ファイル

- `server/domain/endpoint_events.go` — イベント種別の削除
- `server/domain/session_endpoint.go` — readLoop, writeLoop, handleControlEvent, close の変更
- `server/domain/protocol.go` — EncodeLeaveMessage の追加

## 検証

```bash
go test ./server/domain/...
go build ./server/cmd/main.go
```
