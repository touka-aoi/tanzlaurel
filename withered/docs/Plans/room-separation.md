# Plan: Roomの責務分離

## Context

現在のRoomは3つの責務が混在している：
1. セッション管理（sessions map、Join/Leave処理）
2. ゲームループ（60FPS tick、Application.Tick()呼び出し）
3. メッセージルーティング（Broadcast、SendTo、PubSubへの配信）

これをGateway / GameLoop / Matcherに分離し、将来のスケールアウト（Gateway分散、空間インデックス導入）の土台を作る。

**重要**: 今回はまだ単一プロセス。コンポーネント間の境界をインターフェースで切るだけで、NATS導入やプロセス分離は行わない。

## 現状の問題点

- `Room.HandleMessage()`がJoin/Leave/AppDataを全部処理している
- `Room.Broadcast()`がsessions mapを直接走査してPubSub.Publishを呼んでいる
- `Application.OnJoin()/OnLeave()`が存在するのにRoomから呼ばれていない
- `RoomManager.GetRoom()`がSessionEndpointから呼ばれていない（roomIDはプロトコルから直接パース）

## 方針

### Step 1: ApplicationインターフェースにOnJoin/OnLeaveを追加

`server/domain/application.go`を修正。Room.HandleMessage()のJoin/LeaveでApplication.OnJoin()/OnLeave()を呼ぶ。

**変更ファイル:**
- `server/domain/application.go` — OnJoin, OnLeave追加
- `server/domain/room.go` — HandleMessageのJoin/LeaveケースでOnJoin/OnLeave呼び出し

### Step 2: SessionRegistryを導入し、セッション管理をRoomから分離

Roomが直接`sessions map[SessionID]struct{}`を持つのをやめ、SessionRegistryインターフェースに委譲する。

```go
// server/domain/session_registry.go (新規)
type SessionRegistry interface {
    Add(sessionID SessionID)
    Remove(sessionID SessionID)
    List() []SessionID
    Contains(sessionID SessionID) bool
}
```

ローカル実装 `LocalSessionRegistry` も同ファイルに書く（中身はmap）。

**変更ファイル:**
- `server/domain/session_registry.go` — 新規
- `server/domain/room.go` — sessions mapをSessionRegistryに置き換え

### Step 3: BroadcasterをRoomから分離

Room.Broadcast()のロジックを独立させる。

```go
// server/domain/broadcaster.go (新規)
type Broadcaster interface {
    Broadcast(ctx context.Context, data []byte)
    SendTo(ctx context.Context, sessionID SessionID, data []byte)
}
```

ローカル実装 `PubSubBroadcaster` はSessionRegistryとPubSubを持ち、セッション一覧を取得してPublishする。

**変更ファイル:**
- `server/domain/broadcaster.go` — 新規
- `server/domain/room.go` — Broadcast/SendToをBroadcasterに委譲、EnqueueBroadcast/EnqueueSendToも同様

### Step 4: Roomをゲームループ専任にする

Step 1-3の結果、Roomは以下だけになる：
- PubSubからメッセージを受信
- Application.HandleMessage()に委譲
- Application.Tick()を呼ぶ
- Broadcaster経由で配信

```go
type Room struct {
    ID          RoomID
    registry    SessionRegistry
    broadcaster Broadcaster
    application Application
    sendCh      chan roomSend
    tickInterval time.Duration
}
```

**変更ファイル:**
- `server/domain/room.go` — pubsubフィールドを削除、broadcaster/registryに置き換え
- `server/cmd/main.go` — 新しいRoom初期化に合わせてワイヤリング変更

### Step 5: RoomManagerをSessionEndpointで実際に使う

現在SessionEndpointはプロトコルからroomIDを直接パースしているが、RoomManager.GetRoom()を経由するようにする。

**変更ファイル:**
- `server/domain/session_endpoint.go` — handleDataのJoinケースでRoomManager.JoinRoom()を呼ぶ、LeaveでLeaveRoom()を呼ぶ

## 変更ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `server/domain/application.go` | OnJoin/OnLeave追加 |
| `server/domain/session_registry.go` | 新規: SessionRegistry + LocalSessionRegistry |
| `server/domain/broadcaster.go` | 新規: Broadcaster + PubSubBroadcaster |
| `server/domain/room.go` | sessions/pubsub → registry/broadcaster、OnJoin/OnLeave呼び出し |
| `server/domain/room_events.go` | 変更なし（roomSendはBroadcaster内部で使用） |
| `server/domain/session_endpoint.go` | RoomManager.JoinRoom/LeaveRoom呼び出し追加 |
| `server/cmd/main.go` | ワイヤリング変更 |
| `server/domain/mocks/` | mockの再生成 |

## 検証方法

1. `cd withered && go build ./...` — コンパイル通ること
2. `cd withered && go test ./...` — 既存テスト通ること
3. `make dev` — 実際にゲームが動くこと（手動確認）
4. mockの再生成: `go generate ./server/domain/...`
