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