# ECSシステム実装プラン

## Context

`refactor/ecs`ブランチにECSインフラ（World, Scheduler, System interface, Codec）が構築済みだが、7つのSystemの`Run`メソッドがすべてTODO。mainブランチのゲームロジックをECSパターンに移植し、クライアントもECSブランチの新プロトコルに対応させ、Playwrightでスクリーンショットをキャプチャするところまで実装する。

---

## Part 1: サーバーサイド（ECSシステム実装）

### Step 1: Application interfaceの拡張

**`server/domain/application.go`** — OnJoin/OnLeave追加
```go
type Application interface {
    HandleMessage(ctx context.Context, sessionID SessionID, data []byte) error
    Tick(ctx context.Context) ([]byte, error)
    OnJoin(ctx context.Context, sessionID SessionID)
    OnLeave(ctx context.Context, sessionID SessionID)
}
```

**`server/domain/echo_application.go`** — no-opスタブ追加

**`server/domain/room.go`** — HandleMessageのJoin/Leaveケースで`r.application.OnJoin/OnLeave`を呼び出す

### Step 2: PendingInputの型変更

**`server/application/buffer.go`**
```go
type InputEntry struct {
    EntityID EntityID
    KeyMask  uint32
}
```
Push/DrainをInputEntry型に変更。

### Step 3: ShootingWorldの拡張

**`server/application/world.go`** — 追加:
- コンポーネント: `ShootCooldown map[EntityID]int`, `RespawnTimer map[EntityID]int`
- リソース: `SessionToEntity map[domain.SessionID]EntityID`, `EntityToSession map[EntityID]domain.SessionID`, `PendingInputs *PendingInput`, `Static *Static`, `NextEntityID EntityID`
- ヘルパー: `AllocEntity()`, `SpawnActor(sessionID, tag)`, `SpawnBullet(owner, pos, vel)`, `randomPosition()`
- `RemoveEntity`に新コンポーネントのdelete追加

### Step 4: 7つのSystemのRun実装

**`server/application/phase.go`**

| System | ロジック |
|--------|---------|
| RespawnSystem | StateRespawningを走査、RespawnTimer--、0以下でHP=100,StateAlive,ランダム位置 |
| InputMoveSystem | `world.PendingInputs.Drain()` → `keyMaskToDirection` → Position更新 + clamp |
| AutoShootSystem | Player/Bot+Aliveを走査、ShootCooldown--、0以下で最寄り敵にSpawnBullet |
| BulletMoveSystem | TagBullet走査、Position += Velocity、TTL-- |
| CollisionDamageSystem | Bullet vs Actor衝突判定(距離²≤0.64)、被弾→HP-20、死亡→Respawning、命中弾丸→RemoveEntity |
| BulletCleanupSystem | TTL≤0の弾丸をRemoveEntity |
| EncodeBroadcastSystem | 空のまま（Tick内でcodec.EncodeSnapshotが呼ばれるため） |

### Step 5: WitheredApplicationの更新

**`server/application/withered_application.go`**
- `OnJoin`: `world.SpawnActor(sessionID, TagPlayer)`
- `OnLeave`: `SessionToEntity`からEntityID取得 → `RemoveEntity`
- `handleInput`: `ParseInputPayload` → `SessionToEntity`でEntityID解決 → `PendingInputs.Push(InputEntry)`
- コンストラクタ: 新mapの初期化、PendingInputs/StaticをWorldに移動
- app直下の`pendingInputs`フィールド削除

---

## Part 2: クライアント側プロトコル更新

ECSブランチのプロトコルは完全に異なる（varintベースのTransportHeader）。

### 新プロトコル構造（サーバー側既存）

```
Transport: [MsgType(varint)][TotalLen(varint)][Payload]
  MsgType: RoomMessage(0), Ping(1), Pong(2), Assign(3)

RoomMessage Payload: [RoomIDLen(varint)][RoomID(string)][RoomMsgType(varint)][AppPayload]
  RoomMsgType: Join(0), Leave(1), AppData(2)

AppPayload: [PayloadHeader(2bytes)][Data]
  DataType: Input(0), Actor2D(1), ..., Snapshot(5)
```

### Step 6: client/src/protocol.ts 書き換え

- varint読み書き関数追加
- TransportHeader encode/decode
- RoomMessage encode（Join/Leave/AppData）
- ECSスナップショットデコード: Entity bitmask形式 → Actor[]/Bullet[]に変換
  - Tag=Player/Bot → Actor型（EntityID, Position, HP, LifeState）
  - Tag=Bullet → Bullet型（EntityID, Position, Velocity, Owner）
- Assign/Ping/Pong のハンドリング

### Step 7: client/src/game.ts 更新

- `onMessage`: varint TransportHeaderをパース → MsgTypeで分岐
  - Assign → mySessionId保存 + RoomMessage(Join)送信
  - Ping → Pong返信
  - RoomMessage → AppPayload → DataTypeSnapshot → decodeSnapshot
- `gameLoop`: 入力をRoomMessage(AppData)形式で送信
- EntityIDベースの自分識別（myEntityIDをサーバーから受け取る or Assignで暗示）

### EntityID↔自分の識別問題

サーバーはSessionID→EntityIDマッピングを持つが、クライアントにEntityIDを通知する仕組みがまだない。
**解決策**: OnJoin時にサーバーが当該セッションにだけEntityIDを返すメッセージ（JoinAck）を送る。Room.SendToを使う。

### Step 8: renderer.ts 最小限の更新

Actor/Bullet型のフィールドを `sessionId: Uint8Array` → `entityId: number` に変更。`sessionIdEquals` → `entityId === myEntityId` に置き換え。

---

## Part 3: Playwrightスクリーンショット

### Step 9: Playwrightスクリプト作成

**`e2e/screenshot.ts`**（新規）
1. サーバー起動確認（WebSocket接続可能になるまでリトライ）
2. Playwright起動 → `http://localhost:5173` を開く
3. canvas要素が描画されるまで待機（data-connected="true", data-roomJoined="true"）
4. 数秒待機（ゲーム状態が安定するまで）
5. スクリーンショット撮影 → `screenshots/` に保存

---

## 対象ファイル一覧

| ファイル | 変更内容 |
|---------|---------|
| `server/domain/application.go` | OnJoin/OnLeave追加 |
| `server/domain/echo_application.go` | no-opスタブ |
| `server/domain/room.go` | OnJoin/OnLeave呼び出し |
| `server/application/buffer.go` | InputEntry型に変更 |
| `server/application/world.go` | コンポーネント・リソース・ヘルパー追加 |
| `server/application/phase.go` | 7 System Run実装 |
| `server/application/withered_application.go` | OnJoin/OnLeave、handleInput更新 |
| `client/src/protocol.ts` | 新プロトコル対応 |
| `client/src/game.ts` | 新プロトコル対応 |
| `client/src/renderer.ts` | EntityIDベース識別 |
| `e2e/screenshot.ts` | 新規: Playwrightスクリプト |

## 検証方法

1. `cd server && go build ./...` — コンパイル確認
2. `cd server && go test ./...` — テスト
3. `make dev` → ブラウザで接続 → エンティティ移動・射撃・衝突・リスポーン確認
4. Playwrightスクリプト実行 → スクリーンショット撮影
