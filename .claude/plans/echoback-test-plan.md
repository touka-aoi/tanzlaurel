# テスト用main.go作成計画

## 目的
動作確認ができるループバックテスト環境を構築する。

## 設計方針
- **Applicationは呼び出されるだけ** (何にも依存しない)
- Application.Tick()の戻り値をRoomがブロードキャスト
- factoryは作成しない（テスト用main.goで直接初期化）

## 実装内容

### 1. server/domain/room.go の変更

Tick()の戻り値をブロードキャスト（SEND_LOOPの後に追加）。

```go
// ticker処理の最後に追加
if data := r.application.Tick(ctx); data != nil {
    if bytes, ok := data.([]byte); ok {
        r.Broadcast(ctx, bytes)
    }
}
```

### 2. server/domain/echo_application.go の作成

```go
package domain

import "context"

type EchoApplication struct {
    pendingData []byte
}

func NewEchoApplication() *EchoApplication {
    return &EchoApplication{}
}

func (e *EchoApplication) Parse(ctx context.Context, data []byte) (interface{}, error) {
    return data, nil
}

func (e *EchoApplication) Handle(ctx context.Context, event interface{}) error {
    e.pendingData = event.([]byte)
    return nil
}

func (e *EchoApplication) Tick(ctx context.Context) interface{} {
    if e.pendingData == nil {
        return nil
    }
    data := e.pendingData
    e.pendingData = nil
    return data
}
```

### 3. cmd/test/main.go の作成

```go
func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    addr := "localhost:9091"

    // 初期化
    pubsub := domain.NewSimplePubSub()
    defaultRoomID := domain.RoomID("default")
    roomManager := domain.NewSimpleRoomManager(defaultRoomID)
    echoApp := domain.NewEchoApplication()
    room := domain.NewRoom(defaultRoomID, pubsub, echoApp)

    handler := server.Route(pubsub, roomManager)
    srv := server.NewServer(addr, handler)

    // 起動
    go room.Run(ctx)
    go srv.Serve()
    time.Sleep(100 * time.Millisecond)

    // ループバックテスト
    conn, _, _ := websocket.Dial(ctx, "ws://"+addr+"/ws", nil)
    defer conn.Close(websocket.StatusNormalClosure, "done")

    testMessage := []byte("Hello, WebSocket!")
    conn.Write(ctx, websocket.MessageText, testMessage)
    _, received, _ := conn.Read(ctx)

    if string(received) == string(testMessage) {
        slog.Info("LOOPBACK TEST PASSED!")
    } else {
        slog.Error("LOOPBACK TEST FAILED")
    }

    srv.Shutdown(context.Background())
}
```

## 修正が必要なファイル

| ファイル | 変更内容 |
|---------|---------|
| `server/domain/room.go` | Tick()戻り値でBroadcast追加 |
| `server/domain/echo_application.go` | 新規作成 |
| `cmd/test/main.go` | 新規作成 |

## 実装順序

1. `server/domain/room.go` - Tick()戻り値処理追加
2. `server/domain/echo_application.go` - 新規作成
3. `cmd/test/main.go` - 新規作成

## 動作確認

```bash
go run cmd/test/main.go
```

## TODO
- Tick()の戻り値の中身を見てbroadcast/unicastを選べるようにする
- RoomManagerでApplicationを設定する仕組みに移行
