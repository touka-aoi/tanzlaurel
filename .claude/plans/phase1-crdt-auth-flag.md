# Phase 1: CRDT操作への認証フラグ追加

## Context
CRDT操作(Operation)とノードに `authenticated` フラグを追加し、認証状態に基づく削除保護をサーバー・フロントエンド両方で実現する。

## 要件
- Operation/ノードに `authenticated` フィールドを追加
- 認証ユーザー: 全文字を削除可能
- 非認証ユーザー: 非認証文字のみ削除可能（認証文字は削除不可）
- **既存データ（フラグなし）は認証済みとして扱う**
- フロントエンドでも同様の削除制御

## 実装手順

### Step 1: サーバーCRDT構造体
**`web/server/domain/crdt/crdt.go`**
- `Operation` に `Authenticated bool` 追加
- `node` に `authenticated bool` 追加
- `NodeSnapshot` に `Authenticated bool` 追加
- `payloadMsg` に `Authenticated bool` 追加
- `OperationFromPayload()`: JSONから `authenticated` をパース
- `applyInsert()`: ノードに `op.Authenticated` を保存
- `Export()`/`ImportRGA()`: スナップショット永続化対応（既存データは `true` 扱い）
- `IsNodeAuthenticated(id NodeID) bool` メソッド追加

### Step 2: WSプロトコル
**`web/server/handler/ws_protocol.go`**
- `IncomingMessage` に `Authenticated bool` 追加
- `SyncOpMsg` に `Authenticated bool` 追加
- `convertSyncMessage()`: `Authenticated` をマッピング

### Step 3: WSハンドラーの削除バリデーション
**`web/server/handler/ws.go`**
- `handleOp()`: `msg.Authenticated` をpayloadに注入（Phase 2でauth判定を接続）
- `handleOp()`: 非認証opによる認証ノード削除を拒否（projector経由）
  - 暫定: `msg.Authenticated` の値をそのまま使用（Phase 2でサーバーが上書き）

### Step 4: EntryProjector
**`web/server/application/entry_projector.go`**
- `IsNodeAuthenticated(entryID uuid.UUID, nodeID crdt.NodeID) bool` メソッド追加

### Step 5: フロントエンド RGA
**`web/cockpit/src/crdt/rga.ts`**
- `Node` に `authenticated: boolean` 追加
- `Operation` に `authenticated?: boolean` 追加
- `applyInsert`: `op.authenticated ?? true`（デフォルト認証済み = 既存データ互換）
- `isNodeAuthenticated(nodeId): boolean` メソッド追加

### Step 6: フロントエンド SyncManager
**`web/cockpit/src/sync/sync-manager.ts`**
- syncOpから `authenticated` をOperationに伝搬
- `applyTextChange()`: 非認証時、認証ノードの削除をスキップ
- `authenticated` フィールド追加（Phase 2で `auth_status` メッセージから設定）

## 変更ファイル
- `web/server/domain/crdt/crdt.go`
- `web/server/handler/ws_protocol.go`
- `web/server/handler/ws.go`
- `web/server/application/entry_projector.go`
- `web/cockpit/src/crdt/rga.ts`
- `web/cockpit/src/sync/sync-manager.ts`

## 検証
1. `cd web && go test ./...`
2. `cd web/cockpit && npx vitest run`
3. 手動: 非認証opで既存文字（=認証済み扱い）の削除がサーバーで拒否されること
