# アプリケーションレベル Ping/Pong 実装プラン

## Context

サーバーのownerLoopが30秒の`IsIdle`チェックでPongIdleを見ているが、pongを更新する仕組みがないため常に30秒で切断される。
サーバーからpingを送信し、クライアントがpongを返す方式で死活監視の基盤を作る。

## フロー

```
HeartbeatService goroutine (5秒間隔) → ping送信 → writeCh → クライアント
クライアント onMessage → ping検知 → pong送信
Server readLoop → pong受信 → handleControlMessage → evPong → ownerLoop → TouchPong()
```

## スコープ

- HeartbeatServiceは**ping送信のみ**を担当するシンプルなサービス
- idle検知は既存のownerLoopに残す（後で整理）
- pong受信によるTouchPong()は既存のevPong → handleControlEventの仕組みを活用
- `ControlSubTypePing`/`ControlSubTypePong`定数はサーバー側（`protocol.go`）に既存。追加不要

## 変更ファイルと内容

### 1. `server/domain/heartbeat_service.go` (新規)

```go
type HeartbeatService struct {
    pingInterval time.Duration
    session      *Session
    writeCh      chan<- []byte
}

func (h *HeartbeatService) Run(ctx context.Context) {
    // pingInterval間隔でEncodePingMessageをwriteChに送信
    // writeCh fullの場合はdrop + warn log
}
```

### 2. `server/domain/protocol.go`

- `EncodePingMessage(sessionID SessionID) []byte` を追加
  - `EncodeAssignMessage`と同構造、`ControlSubTypePing`を使用

### 3. `server/domain/session_endpoint.go`

- `handleControlMessage`: `ControlSubTypePong` ケースを追加
  ```go
  case ControlSubTypePong:
      se.sendCtrlEvent(ctx, endpointEvent{kind: evPong})
  ```
- `Run()`: **Assignメッセージ送信後に**HeartbeatServiceのgoroutineを起動（ctxで管理）
  - アプリケーションレベルのheartbeatなので、セッション確立後に開始
  - Assign前に起動するとクライアントがsessionIdを持っておらずpongを返せない

### 4. `client/src/protocol.ts`

- 定数追加: `CONTROL_SUBTYPE_PING = 4`, `CONTROL_SUBTYPE_PONG = 5`

### 5. `client/src/game.ts`

- `onMessage`でpingを受信 → 即座にpong返信
  ```typescript
  if (subType === CONTROL_SUBTYPE_PING) {
      const pongMsg = encodeControlMessage(this.mySessionId, this.seq++, CONTROL_SUBTYPE_PONG);
      this.ws.send(pongMsg);
  }
  ```

## 検証方法

1. `go test ./server/domain/...` でテストパス
2. HeartbeatServiceの単体テスト（pingがwriteChに送信されることを確認）
3. サーバー + クライアント接続で30秒以上接続維持を確認
4. サーバーログでping送信/pong受信のデバッグログが出ること
