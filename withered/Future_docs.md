# Future Architecture: 1万人同時接続リアルタイムワールド

## ゴール

- 1万人が1つの論理ワールドに同時接続
- 30-100msのレイテンシ
- メタバース基盤として拡張可能

## コンポーネント分離

現在のRoomは3つの責務が混在している（セッション管理、ゲームループ、メッセージルーティング）。
これを3つの独立コンポーネントに分離する。

```
┌──────────┐       ┌──────────────────┐       ┌──────────┐
│ Gateway  │       │   Game Server    │       │ Matcher  │
│ ×N台     │◄─────►│                  │◄─────►│          │
│          │  NATS │                  │       │          │
│ WT接続    │       │  Uniform Grid    │       │ 配置決定  │
│ Session  │       │  LOD配信          │       │ 負荷分散  │
│ Heartbeat│       │  ECS (1 World)   │       │          │
└──────────┘       └──────────────────┘       └──────────┘
```

### Gateway (現SessionEndpoint相当)
- WebTransport接続の受付・維持
- Session/Heartbeat管理
- Game Serverへの入力転送、クライアントへのスナップショット配信
- ステートレス。水平スケール可能

### Game Server (現Room + Application相当)
- ECS Worldを保持し60FPS tickを回す
- 空間インデックス(Uniform Grid)で近傍計算
- LODに基づくスナップショット生成
- 1プロセスで1万人を処理（空間インデックスによりO(N²)を回避）

### Matcher (現RoomManager相当)
- プレイヤーのワールド配置を決定
- 将来的にSector分割時のロードバランシング

## トランスポート: WebTransport

WebSocketからWebTransportへ移行する。

| 特性 | WebSocket | WebTransport |
|------|-----------|-------------|
| プロトコル | TCP | QUIC (UDP) |
| HoLブロッキング | あり | なし |
| unreliable配信 | 不可 | datagram対応 |
| 既存varint形式との親和性 | 関係なし | QUIC varint形式をそのまま利用可 |

既存プロトコルがQUIC varint (RFC 9000) のエンコーディング形式を採用済みのため、WebTransport(QUIC)上でも同じバイナリプロトコルをそのまま使える。

### チャネル使い分け
- **unreliable datagram**: 位置情報、弾の状態（古いパケットは不要）
- **reliable stream**: Join/Leave、インベントリ操作、チャット（確実に届ける必要あり）

## 空間インデックス: Uniform Grid

ワールド(100×100)をセルサイズ5で区切り、20×20=400セルのGridを構築する。

```
操作       計算量
─────────────────
挿入       O(1)    Grid[x/cellSize][y/cellSize]に追加
移動       O(1)    旧セルから削除、新セルに追加
近傍検索   O(K)    周囲セルを走査（K=近傍Entity数）
```

Quadtree/Spatial Hashと比較して、固定サイズワールド+毎tick全Entity移動のケースで最もシンプルかつ高速。

## LOD (Level of Detail) 配信

3DCGのLODと同じ発想。セルのチェビシェフ距離でLODレベルを決定し、距離に応じて配信頻度とデータ量を削減する。

```
LODレベル   距離(セル)   頻度      データ           帯域/人
──────────────────────────────────────────────────────
LOD 0      ≤2          60fps    全フィールド       48 KB/s
LOD 1      ≤6          20fps    位置+HP           24 KB/s
LOD 2      ≤12         5fps     位置のみ           4 KB/s
カリング    >12         なし     配信しない          0
```

**Entity単位の距離計算は不要**。セル座標の差がそのままLODレベルになる。

### 帯域見積もり

```
1プレイヤーあたり: ~76 KB/s
全体: 76 KB × 10,000人 = 760 MB/s
Gateway 4台に分散: 190 MB/s/台
```

単純AOI(全部60fps)の600MB/sと比較して約1/8。

## ステート管理

### 揮発ステート（ECS World、メモリ上）
- Entity位置、弾、エフェクト、HP
- Game Serverプロセスのメモリに保持
- プロセスが死んだら消える → クライアントに再接続を促す

### 永続ステート（将来、DB）
- アバター、インベントリ、配置オブジェクト
- ECS Worldには入れない
- Game Server起動時にDBから読み込み、定期的に書き戻す
- tickループ内でDBアクセスはしない

## スケール戦略

```
段階          対象人数     手法
──────────────────────────────────────────────
現在          ~100        単一プロセス、空間インデックスなし
Phase 1       ~10,000     空間インデックス + LOD + Gateway分散
Phase 2       ~100,000    空間分割(Sector)、各Sectorが独立ECS World
```

Phase 2（Sector分割）は現時点では設計しない。
Phase 1で1万人をカバーし、必要になったら空間分割を導入する。

## 未決定事項

- ブラウザ対応範囲（WebTransportはChrome系のみ、Safari/Firefox未対応）
- コンポーネント間メッセージング（NATS vs 他）
- Sector分割の要否と設計（Phase 2）
- 永続ストアの選定（PostgreSQL / Redis / 他）
