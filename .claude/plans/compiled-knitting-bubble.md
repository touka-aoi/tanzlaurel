# シューティング + HP + AIボット実装計画

## Context

現在のゲームはWASDでキャラ（丸）が動くだけのシンプルな状態。
これに「自動射撃」「HP制」「ルールベースAIボット」を追加し、擬似マルチプレイのシューティングゲームにする。

## 設計方針

| 項目 | 方針 | 理由 |
|------|------|------|
| 射撃 | サーバーTick()内で自動射撃 | 決定論的。AI/人間で同じロジック |
| 弾丸 | Field内の別エンティティ。アクターと一括ブロードキャスト | ネットワーク効率。Tick1回で1メッセージ |
| AIボット | Application層に直接注入 | WebSocket不要。プロトコルオーバーヘッドなし |
| ブロードキャスト | アクター26bytes + 弾丸20bytes の結合フォーマット | 既存の拡張で対応可能 |

## ゲームパラメータ

- HP: 100, 弾丸ダメージ: 20 (5発で死亡)
- 射撃間隔: 30tick (0.5秒 @60FPS)
- 弾速: 3.0 unit/tick, TTL: 120tick (2秒)
- リスポーン: 180tick (3秒)
- ボット数: 3体 (デフォルト)
- 当たり判定: 円同士 (弾丸半径0.3, アクター半径0.5)

## 実装フェーズ

### Phase 1: HP・状態システム (サーバー + クライアント)

**サーバー側**

1. **`server/application/field.go` - Actor拡張**
   - Actor に `HP uint8`, `State ActorState`, `shootCooldown int`, `respawnTimer int` 追加
   - `ActorState` bitmask (1バイト内でビット領域を分離):
     - bit 0-3: 状態フラグ — `StateAlive=0x01`, `StateRespawning=0x02`
     - bit 4-7: 種別フラグ — `KindPlayer=0x00`, `KindBot=0x10`
   - `SpawnAtCenter` で HP=100, State=ActorAlive に初期化
   - `DamageActor(sessionID, damage)`, `TickRespawns()`, `IsAlive()` メソッド追加

2. **`server/application/withered_application.go` - エンコーディング変更**
   - `encodeActorPositions`: actorSize を 24→26 bytes に変更
   - 各アクターに HP(u8) + State(u8) を追加エンコード

**クライアント側**

3. **`client/src/protocol.ts` - Actor型・デコード更新**
   - `Actor` interface に `hp: number`, `state: number` 追加
   - 状態フラグ定数: `STATE_ALIVE=0x01`, `STATE_RESPAWNING=0x02`
   - 種別フラグ定数: `KIND_PLAYER=0x00`, `KIND_BOT=0x10`
   - ヘルパー: `isAlive(state)`, `isBot(state)` でビットマスク判定
   - `decodeActorBroadcast`: ACTOR_SIZE を 24→26、HP/State デコード追加

4. **`client/src/renderer.ts` - HPバー・死亡表示**
   - アクター上部にHPバー描画
   - 死亡時は半透明 (globalAlpha=0.3)
   - Bot は赤色 (`#f87171`)、人間は青 (`#60a5fa`)、自分は緑 (`#4ade80`)

### Phase 2: 弾丸システム

**サーバー側**

5. **`server/application/bullet.go` - 新規作成**
   - `Bullet` struct: `ID(u32)`, `OwnerID`, `Position`, `Velocity`, `TTL`
   - 定数: `BulletSpeed`, `BulletTTL`, `BulletRadius`, `ActorRadius`, `ShootCooldown`
   - `HitEvent` struct: `BulletID`, `VictimID`, `AttackerID`

6. **`server/application/field.go` - 弾丸管理追加**
   - `Bullets []*Bullet`, `nextBulletID` フィールド追加
   - `AddBullet()`: 弾丸生成
   - `TickBullets()`: 位置更新 + TTL減算
   - `CheckBulletCollisions()`: 円-円判定 → HitEvent返却 + 命中弾除去
   - `RemoveExpiredBullets()`: TTL <= 0 の弾丸除去

7. **`server/application/withered_application.go` - Tick拡張**
   - `processAutoShoot()`: 全生存アクターが最寄り敵に自動発射
   - Tick順序:
   - 0. リスポーン処理 (復活後一定時間無敵)
     1. 人間入力適用 (移動)
     3. 自動射撃
     4. 弾丸移動
     5. 衝突判定 → ダメージ適用
     6. 期限切れ弾丸除去
     7. エンコード → ブロードキャスト
   - `encodeGameStateBroadcast(actors, bullets)`:
     `[ActorCount(u16)][Actors...][BulletCount(u16)][Bullets...]`

**クライアント側**

8. **`client/src/protocol.ts` - Bullet型・デコード追加**
   - `Bullet` interface: `id, ownerSessionId, x, y, vx, vy`
   - `decodeActorBroadcast` → `decodeGameState` に改名
   - アクター後にBulletCountとBulletデータをデコード

9. **`client/src/game.ts` - 弾丸状態・補間**
   - `bullets: Bullet[]`, `lastBulletUpdate: number` 追加
   - gameLoopで弾丸位置を速度で補間（滑らかな描画）

10. **`client/src/renderer.ts` - 弾丸描画**
    - 黄色の小円 (半径3px) で弾丸描画
    - `render()` に bullets パラメータ追加

### Phase 3: AIボットシステム

**サーバー側**

11. **`server/application/bot.go` - 新規作成**
    - `BotController` interface: `Decide(self, allActors, allBullets) BotAction`
    - `BotAction`: `MoveDirection Position2D`
    - `BotInstance`: `SessionID`

12. **`server/application/bot_rule.go` - 新規作成**
    - `RuleBotController` 実装:
      - 近距離: 後退 (逃げ)
      - 中距離: 横移動 (ストレイフ)
      - 遠距離: 接近 (追跡)
      - 被弾回避: 接近する弾丸から逃げる

13. **`server/application/withered_application.go` - ボット統合**
    - `bots map[SessionID]*BotInstance`, `botController BotController` フィールド追加
    - `AddBot()` / `RemoveBot()` メソッド
    - `NewWitheredApplication(botCount int)` でボット数を受け取る
    - Tick内で `processBotDecisions()` を人間入力適用後に実行

14. **`server/cmd/main.go` - ボット数設定**
    - `NewWitheredApplication(3)` でデフォルト3体起動

## ワイヤーフォーマット

### アクター (26 bytes/actor)
```
SessionID [16]byte | X f32 | Y f32 | HP u8 | State u8
```

### 弾丸 (36 bytes/bullet)
```
BulletID u16 | OwnerID [16]byte | X f32 | Y f32 | VX f32 | VY f32
```

### ブロードキャスト全体
```
[Header(25)] [PayloadHeader(2)] [ActorCount(u16)] [Actors...] [BulletCount(u16)] [Bullets...]
```

## 変更ファイル一覧

### 新規作成
- `server/application/bullet.go`
- `server/application/bot.go`
- `server/application/bot_rule.go`

### 既存変更
- `server/application/field.go` - Actor拡張, Bullet管理, 衝突判定
- `server/application/withered_application.go` - Tick拡張, 射撃, ボット, エンコード
- `server/cmd/main.go` - ボット数パラメータ
- `client/src/protocol.ts` - Actor/Bullet型, デコード更新
- `client/src/game.ts` - 弾丸状態, 補間
- `client/src/renderer.ts` - HPバー, 弾丸, Bot色分け

### 変更不要
- `server/domain/protocol.go` - KeyMask/Headerは変更なし
- `server/domain/room.go` - Tick/Broadcastの仕組みはそのまま
- `client/src/input.ts` - 入力は変更なし (射撃は自動)

## 検証方法

1. **Phase 1完了後**: サーバー起動 + クライアント接続 → HPバーが表示される
2. **Phase 2完了後**: 2つのブラウザタブで接続 → 自動射撃・弾丸飛翔・HP減少・死亡リスポーンを確認
3. **Phase 3完了後**: 1つのブラウザタブで接続 → 赤色のBotが3体出現し、移動・射撃してくる
4. **テスト**: `go test ./server/application/...` で既存テスト + 新規テストが通ること
