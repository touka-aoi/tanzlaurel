# 開発進捗・トラブルシューティング

## 2026-02-28: CRDTブログサービス Phase 1〜4 実装完了

### 実装済み

| Phase | 内容 | コミット数 |
|-------|------|-----------|
| Phase 1 | RGA CRDTコアロジック + PBTテスト (Go) | 1 |
| Phase 2 | domain層、インメモリアダプター、ロガー、REST API、サーバー構築 | 5 |
| Phase 3 | SyncService、WebSocket、TS RGA、統合テスト | 4 |
| Phase 4 | Preact + TailwindCSS v4 フロントエンド | 1 |

### テスト状況

- Go PBT (rapid): RGA収束性・親子順序・削除 — 全合格
- Go ユニット: EventStore/EntryStore/Handler/SyncService/WebSocket — 全合格 (18件)
- TS vitest: RGA収束性・冪等性・pending — 全合格 (6件)

### 未実装 (Phase N, Phase 5)

- ScyllaDB導入 (現在はインメモリ)
- オフライン編集対応 (IndexedDB)
- エディタのテキスト差分→CRDTオペレーション変換

---

## 2024-02-01: Vite + TypeScript で型インポートエラー

### エラー内容

```
renderer.ts:3 Uncaught SyntaxError: The requested module '/src/protocol.ts' does not provide an export named 'Actor' (at renderer.ts:3:10)
```

### 原因

TypeScriptの`interface`や`type`は**ランタイムに存在しない**（コンパイル時のみ存在する）。

```typescript
// protocol.ts
export interface Actor {
  sessionId: bigint;
  x: number;
  y: number;
}
```

```typescript
// renderer.ts - 問題のあるコード
import { Actor } from "./protocol";
```

Viteはブラウザで直接TypeScriptを実行する際、型情報を削除する。しかし通常の`import`だと、ブラウザが「`Actor`という値がexportされているはず」と期待してエラーになる。

### 解決方法

`import type`を使用して「これは型だけのimport」と明示する。

```typescript
// renderer.ts - 修正後
import type { Actor } from "./protocol";
```

`import type`は「このimportはコンパイル時に完全に消える」ことを明示するので、Viteがブラウザ向けにトランスパイルする際に正しく処理できる。

### 予防策

`tsconfig.json`で以下を設定すると、型のみのimportに`import type`を強制できる：

```json
{
  "compilerOptions": {
    "verbatimModuleSyntax": true
  }
}
```

---

## 2026-02-03: SessionEndpointとRoomの責務分離

### 背景

`session_endpoint.go`の設計について相談。現状はすべてのメッセージがPubSubを通じてRoomに流れていた。

### 懸念1: joinRoomなどの制御メッセージの処理場所

**問題**: Join/LeaveなどのメッセージもRoomで処理しているが、これはサーバーループ(SessionEndpoint)で処理すべきではないか。

**解決**: SessionEndpointで制御メッセージをハンドリングし、RoomManager経由でJoin/Leaveを処理する設計に変更。

```
SessionEndpoint.readLoop
    ├─ Control (Join/Leave) → RoomManager.JoinRoom/LeaveRoom
    └─ Application msg → PubSub → Room
```

### 懸念2: Joinのレスポンスをどう返すか

**問題**: Join処理の結果をクライアントにどう返すか。

**検討した案**:
1. readLoopで同期的に処理 → シンプルだがownerLoopとの一貫性が崩れる
2. ownerLoop経由 → ADR-002に忠実だが複雑
3. リクエストIDで紐付け → プロトコル設計が複雑

**解決**: `writeCh`に返す。既存のwriteLoop/Send()を活用し、レスポンスも通常メッセージも同じ経路で統一。

### 懸念3: readLoopでの状態変更とrace condition

**問題**: readLoopで`joined`フラグを変更するとownerLoopとのrace conditionの可能性。

**解決**: `joined bool`フラグは不要。`roomID`の有無で状態を判定する。
- `roomID == ""` → 未Join
- `roomID != ""` → Join済み

### 懸念4: サーバーループとRoomで2回パースが発生する

**問題**: SessionEndpointでDataType/SubTypeを判定するためにパースし、Room/Applicationでも同じデータを再度パースする。

**解決**: 将来的にSessionEndpoint(接続サーバー)とRoom(ゲームサーバー)が別プロセスに分離する可能性を考慮。疎結合を維持するため、2回パースは許容するトレードオフとして受け入れる。生データ(バイト列)を渡す方がネットワーク境界が明確。

### 懸念5: PayloadHeader.SubTypeがControlSubType固定

**問題**: `PayloadHeader.SubType`が`ControlSubType`型で固定されているが、SubTypeの解釈はDataTypeによって変わる（Actor→ActorSubType, Control→ControlSubType）。

**解決**: `SubType`を`uint8`に変更。使用側でDataTypeに応じてキャストする。`docs/architecture/protocol_subtype.md`にドキュメント化。

### 懸念6: JoinPayloadの形式が未定義

**問題**: `ControlSubTypeJoin`は定義されているが、Joinメッセージのペイロード(roomIDをどう送るか)が未定義。

**検討した案**:
1. 固定長roomID (16バイトUUID)
2. 可変長roomID (length prefix)
3. デフォルトルームのみ (ペイロードなし)

**解決**: 16バイト固定(UUID形式)を採用。`protocol.md`と`protocol.go`に追記。

```
JoinPayload (16 bytes):
┌──────────────────────────────┐
│        roomID (16B)          │
└──────────────────────────────┘
```

### 実装した変更

1. **SessionEndpoint.handleData** - Header/PayloadHeaderをパースし、DataTypeで分岐
2. **SessionEndpoint.handleControlMessage** - Join/Leaveを処理、RoomManager経由
3. **ParseJoinPayload** - 16バイトのRoomIDをパース
4. **RoomManagerインターフェース** - `JoinRoom`/`LeaveRoom`メソッド追加
5. **SimpleRoomManager** - `JoinRoom`/`LeaveRoom`実装
6. **readLoop** - `handleData`を呼ぶように変更

### 関連ドキュメント

- `docs/architecture/protocol.md` - JoinPayload定義追加
- `docs/architecture/protocol_subtype.md` - SubType設計の説明（新規作成）

---

## 2026-03-01: サーバー側RGA管理・Entry永続化・Markdown出力を実装

### 背景

SyncServiceはopのバイト列をEventStoreに保存・配信するだけで、サーバー側RGAを構築していなかった。
Entry.Text/Title/Contentも空のまま。OSOT(SyncService)とProjector(EntryProjector)の責務を分離した。

### 実装した変更

- `domain/crdt/crdt.go`: `OperationFromPayload` (Payload→Operation変換)、`Export`/`ImportRGA` (RGAスナップショット)
- `application/entry_projector.go`: RGA管理 + Entry更新 + Markdown出力
- `adapter/jsonfile/`: JSONファイルベースのEventStore/EntryStore/RGAStateStore
- `handler/ws.go`: Broadcast後にProjector呼び出し
- `cmd/main.go`: jsonfileアダプタ + EntryProjector注入 + 起動時Restore
- `web/Makefile`: `make dev`/`make stop`/`make restart`

### 懸念1: RGAスナップショットからpendingが消失

**問題**: RGA.Applyでseenに登録された後にafterノード未到着でpendingに入ったopが、Export時に保存されず、Import後に冪等で弾かれてノードが永久に失われた。

**解決**: `RGASnapshot.Pending`フィールドを追加し、Export/Importでpendingも永続化するようにした。

---

## 2026-03-07: インフラ構築 (Docker Compose + Cloudflare Tunnel + Ansible)

### 実装した変更

- `web/Dockerfile`: Go multi-stage build (Go 1.26 → distroless)
- `web/cockpit/Dockerfile`: Node build → Nginx配信
- `web/docker-compose.yml`: app + nginx + cloudflaredの3コンテナ構成
- `web/infra/nginx/default.conf`: 静的ファイル配信 + APIリバースプロキシ(WebSocket対応) + SPA fallback
- `web/infra/terraform/`: Cloudflare Tunnel + DNS設定 (main.tf, variables.tf, outputs.tf)
- `web/infra/ansible/`: VM自動セットアップ (docker, cloudflared, app roles)

### アーキテクチャ

```
Internet → Cloudflare Tunnel → nginx(:80) → app(:8080)
                                  ↓
                          静的ファイル (SPA)
```

---

## 2026-03-31〜04-01: Roomの責務分離と1万人同時接続に向けた設計・負荷テスト

### 背景

Roomの設計に違和感があり、将来1万人同時接続（メタバース基盤）を見据えてアーキテクチャを見直した。Roomが「セッション管理」「配信」「ゲームループ」の3責務を持っており、Gateway分散時のボトルネックになる構造だった。

### 懸念1: Roomの責務が大きすぎる

**問題**: Roomがsessions map管理、Broadcast、60FPS tickループを全て担当。コンポーネントとしてApplication/Room/Matchingが分離されていない。

**解決**: SessionRegistry（セッション管理）とBroadcaster（配信ロジック）をインターフェースとして分離し、Roomをゲームループ専任にリファクタ。将来PubSubをNATSに差し替えるだけでプロセス分離可能な設計とした。

### 懸念2: 1万人接続時のTick計算量

**問題**: AutoShootSystemがO(N²)の全探索。1万人で1億回/tickとなり16msに収まらない。

**解決**: Uniform Grid（空間インデックス）の導入で O(N×K) に削減する方針。100×100マップをセルサイズ5で20×20=400セルに分割。近傍検索はO(K)。実装は負荷テスト後に実施予定。

### 懸念3: 1万人接続時のBroadcast帯域

**問題**: 全EntityのSnapshotを全員に60fps送信すると120GB/s。現実的でない。

**解決**: LOD（Level of Detail）配信を設計。セルのチェビシェフ距離でLODレベルを決定し、距離に応じて配信頻度とデータ量を削減。Entity単位の距離計算不要。帯域は約1/8に削減見込み。

### 懸念4: ゲームステートの持ち方

**問題**: 1万人の位置情報等をどこに保存するか。メモリだけでは不足するのでは。

**解決**: 計算の結果、1万人×44bytes=440KB、弾含めても3MB程度でメモリは問題なし。ボトルネックは「保存」ではなく「計算」と「配信」。揮発ステート（位置等）はメモリ、永続ステート（アバター等、将来）はDB。tickループ内でDBアクセスはしない。

### 懸念5: トランスポート選定

**問題**: WebSocketはTCPベースでHoLブロッキングがある。位置情報等は古いパケットを捨てたい。

**解決**: WebTransportを将来のトランスポートとして選定。unreliable datagram（位置情報）とreliable stream（Join/Leave等）の使い分け。既存プロトコルがQUIC varint形式を採用済みで親和性が高い。

### 懸念6: 耐障害性

**問題**: Game Serverが死んだらステートが消える。

**解決**: シューティングゲームの揮発ステート（位置・弾）は消えても再接続で復帰可能。メタバースの永続データ（アバター・インベントリ）はDBに保存し、tickループ外で定期的に書き戻す設計。Raftは60fps tickには不向き。まず「死んだら再接続」で割り切り、将来Active-Standby（決定的シミュレーション）を検討。

### 実装した変更

1. **SessionRegistry** (`server/domain/session_registry.go`) - セッション管理をRoomから分離
2. **Broadcaster** (`server/domain/broadcaster.go`) - 配信ロジックをRoomから分離
3. **Room リファクタ** (`server/domain/room.go`) - registry/broadcaster委譲、OnJoin/OnLeave呼び出し追加
4. **Applicationインターフェース** (`server/domain/application.go`) - OnJoin/OnLeave追加
5. **Botロードテスト** (`server/cmd/bot/main.go`) - 1万体接続テスト用Bot実装
6. **room_events.go** - 未使用のroomCtrl型を削除
7. **main.go** - ログレベルをWarnに変更、新コンポーネントのワイヤリング

### 負荷テスト結果（暫定）

- 100体: 問題なし
- 1000体: 問題なし
- 1600体前後: Bot側が一斉にconnection reset。Snapshotの帯域が原因と推定
- **次のステップ**: Broadcast頻度の削減（60fps→20fps）、その後Grid + LOD導入

### 関連ドキュメント

- `withered/Future_docs.md` - 1万人同時接続アーキテクチャ設計
- `withered/docs/Plans/room-separation.md` - Room分離プラン

---

## TODO

| # | タスク | 備考 |
|---|--------|------|
| 1 | ScyllaDBへの移行 | 現在はJSONファイル永続化。CDC経由でProjectorを非同期化。ライブラリ: [gocqlx](https://github.com/scylladb/gocqlx) |
| 2 | ~~デプロイ環境構築~~ | ✅ Docker Compose + Cloudflare Tunnel + Ansible |
| 3 | フロントのデザイン整理 | UI/UXの改善 |
| 4 | 認証/認可 + 管理画面 | ログイン機能。エントリごとに公開版(誰でも編集可)と原文(ログインユーザーのみ編集可)の2状態を保持。閲覧時にトグルで切替可能。荒らし対策として原文をいつでも表示・復元できる |