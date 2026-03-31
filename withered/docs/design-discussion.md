# 設計議論メモ (2026-03-31〜04-01)

## 1. 接続のスケール

### 現状の構造

```
Client ──WS──► SessionEndpoint ──PubSub(メモリ)──► Room ──► Application
```

1プロセスに全部入り。SessionEndpointが接続ごとにgoroutine群（readLoop/writeLoop/subscribeLoop/ownerLoop/heartbeat）を起動。

### Gateway分離の設計

SessionEndpoint（接続管理）とRoom（ゲームループ）をプロセス分離する。

```
現在: 1プロセスに全部
┌──────────────────────────────────────┐
│ SessionEndpoint + PubSub + Room      │
└──────────────────────────────────────┘

将来: 2プロセス
┌─ Gateway Pod ─┐    ┌─ Game Server ──┐
│ SessionEndpoint│◄──►│ Room           │
│ WS/WT接続管理   │NATS│ Application    │
└───────────────┘    └────────────────┘
```

変わるのはPubSubの実装だけ。SessionEndpointは今も`PubSub.Publish("room:xxx", msg)`でRoomと通信しており、Roomを直接知らない。PubSubをNATSに差し替えればプロセス分離できる。

### Roomの責務分離（実装済み）

Roomから「セッション管理」と「配信」を抜き出し、ゲームループ専任にした。

```
SessionRegistry  ← 誰がいるか管理（Add/Remove/List）
Broadcaster      ← 配信（内部でSessionRegistry.List() + PubSub.Publish）
Room             ← ゲームループだけ（registry/broadcaster経由で操作）
```

SDK利用者が触るのはApplicationインターフェースだけ。SessionRegistry/BroadcasterはSDK内部。

### Broadcast帯域の問題

```
1600体 × 全Entity snapshot × 60fps → サーバーのwriteが追いつかない → 接続切断
```

負荷テストで1600体前後が限界であることを確認。帯域削減にはLOD配信が必要。

### トランスポート: WebTransport

WebSocketからWebTransportへの移行を予定。

- unreliable datagram: 位置情報（古いパケットは不要、HoLブロッキング回避）
- reliable stream: Join/Leave、チャット（確実に届ける必要あり）
- 既存プロトコルがQUIC varint (RFC 9000)のエンコーディング形式を採用済みで親和性が高い
- WebRTCと比較してSTUN/TURN不要、サーバー権威型に適する

## 2. 空間分割

### 空間インデックス vs 空間分割（Sector）

2つの異なる概念がある。

**空間インデックス（Uniform Grid）**: 1つのECS World内でEntityの近傍検索を高速化するデータ構造。

```
ワールド100×100 をセルサイズ5で区切る → 20×20 = 400セル

操作       計算量
挿入       O(1)
移動       O(1)    旧セルから削除、新セルに追加
近傍検索   O(K)    周囲セルを走査
```

**空間分割（Sector）**: ワールドを物理的に複数プロセスに分割。各Sectorが独立したECS Worldを持つ。

```
┌────────┬────────┐
│Sector0 │Sector1 │  各プロセスが領域を担当
│ GS-0   │ GS-1   │  境界のEntity受け渡しが複雑
├────────┼────────┤
│Sector2 │Sector3 │
│ GS-2   │ GS-3   │
└────────┴────────┘
```

### 結論: まずGridで十分

1万人なら空間インデックス（Grid）で1プロセスで処理可能。Sector分割は10万人〜で検討。

Gridがあれば計算も配信も同じ空間インデックスで近傍に絞れる。プレイヤー移動時もセルの所属が変わるだけで、Sectorのような境界ハンドオフ問題が発生しない。

### LOD（Level of Detail）配信

3DCGのLODと同じ発想。距離に応じて配信頻度とデータ量を削減。

```
LOD 0 (近距離 ≤2セル):   60fps, 全フィールド
LOD 1 (中距離 ≤6セル):   20fps, 位置+HP
LOD 2 (遠距離 ≤12セル):  5fps,  位置のみ
カリング (>12セル):       配信しない
```

Entity単位の距離計算は不要。**セルの座標差（チェビシェフ距離）がそのままLODレベル**になる。

```go
func lodForCell(playerCol, playerRow, cellCol, cellRow int) LODLevel {
    dc := abs(playerCol - cellCol)
    dr := abs(playerRow - cellRow)
    dist := max(dc, dr)
    switch {
    case dist <= 2:  return LOD0
    case dist <= 6:  return LOD1
    case dist <= 12: return LOD2
    default:         return LODCull
    }
}
```

帯域削減効果: 単純AOI(全部60fps)の約1/8。

### Ghost Entity（将来、Sector分割時のみ）

Sector境界付近で隣のSectorのEntityを見るための読み取り専用コピー。MMOのエリア移動（ローディング画面）を使わずシームレスにするために必要。現時点では不要。

## 3. キューとI/O

### キュー（PubSub）の役割

I/Oの総量は減らない。キューの役割はTickの安定性を確保すること。

```
キューなし:
  tick {
    Read  ← ネットワーク遅延で詰まる可能性
    処理
    Write ← 1万人に書くと16ms超える可能性
  }
  → tickの間隔がI/Oに引きずられてブレる

キューあり（現在の設計）:
  tick {
    キューから取る  ← メモリ操作、一瞬
    処理
    キューに入れる  ← メモリ操作、一瞬
  }
  → tickは常に安定。I/Oは別goroutineが非同期で処理
```

### 現在の設計はゲームサーバーのI/O分離パターン

Webサーバー的な名前だが、やっていることはゲームサーバーのI/O分離パターンそのもの。

```
SessionEndpoint (readLoop/writeLoop)  = I/Oスレッド群
PubSub                                = 入力/出力キュー
Room.Run()                            = Tickスレッド
```

### 各最適化の役割分担

```
手法              効果
─────────────────────────────────
キュー/Gateway分離  Tickを安定させる（I/O量は変わらない）
LOD               I/O量そのものを減らす
Grid              Tick計算量を減らす
```

### ステート管理

メモリは問題にならない。1万人×44bytes=440KB、弾含めても3MB程度。

```
揮発ステート（ECS World、メモリ上）:
  位置、弾、エフェクト、HP → プロセスが死んだら消える → 再接続

永続ステート（将来、DB）:
  アバター、インベントリ、配置物 → tickループ外で定期書き戻し
```

### 耐障害性

ゲームサーバーはステートがメモリ上にあるため、Webサービスのように単純な水平スケールができない。

```
小規模（~数百人）:  死んだら再接続。割り切り。← 今ここ
中規模（MMO等）:   Active-Standby + 決定的シミュレーション
大規模:           空間分割 + 各Sectorごとにフェイルオーバー
```

Raftは60fps tickには不向き（合意のラウンドトリップでレイテンシ予算を使い切る）。ゲームステートの耐障害性にはスナップショット+リプレイの方が現実的。
