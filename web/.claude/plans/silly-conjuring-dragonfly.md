# Draftビュー関連の完全削除

## Context
AuthEntryProjector（原文/Draft表示）は認証済みopsのみのテキストを提供していたが、
サーバー・クライアント両方で非認証ユーザーの削除をブロックする仕組みが入ったため不要になった。
Draft関連の全コードを削除して構造をシンプルにする。

## 削除対象ファイル（丸ごと削除）
- `web/server/application/auth_entry_projector.go`
- `web/server/domain/auth_entry.go`
- `web/server/adapter/jsonfile/auth_entry_store.go`
- `web/server/adapter/jsonfile/auth_rga_state_store.go`
- `web/server/adapter/memory/auth_entry_store.go`
- `web/server/adapter/memory/auth_rga_state_store.go`

## 修正対象ファイル

### 1. `web/server/cmd/main.go`
- AuthEntry系の初期化ブロック削除（authEntryStore, authRGAStateStore, authProjector）
- authProjector.Restore呼び出し削除
- NewRouter引数からauthEntryStore, authProjector削除

### 2. `web/server/router.go`
- NewRouter引数からauthEntryStore, authProjector削除
- handler.NewEntry引数からauthEntryStore削除
- handler.NewWS引数からauthProjector削除
- import不要なら整理

### 3. `web/server/handler/entry.go`
- Entry構造体からauthStoreフィールド削除
- NewEntry引数からauthStore削除
- Get内のoriginalText取得ロジック削除
- EntryDetailResponseからOriginalTextフィールド削除

### 4. `web/server/handler/ws.go`
- WS構造体からauthProjectorフィールド削除
- NewWS引数からauthProjector削除
- handleOp内のauthProjector.Apply呼び出し削除

### 5. `web/server/application/entry_projector_test.go`
- mockAuthEntryStore、newMockAuthEntryStore削除
- TestAuthEntryProjector_*テスト3件削除（AnonDeleteCannotDeleteAuthNodes, AuthDeleteDeletesAuthNodes, AnonInsertExcluded）

### 6. `web/cockpit/src/pages/entry-page.tsx`
- ViewMode型、viewModeステート、originalTextステート削除
- 原文テキスト取得useEffect削除
- Live/Draftトグルボタン削除
- displayText → text に簡素化
- getAuthHeaders import削除

### 7. `web/cockpit/src/components/feed-card.tsx`
- ViewMode型、viewModeステート、originalTextステート削除
- 原文テキスト取得useEffect削除
- Live/Draftトグルボタン削除
- rawContent → liveText に簡素化
- getAuthHeaders import削除

### 8. `web/cockpit/e2e/auth-dual-view.spec.ts`
- TC1: DraftボタンのtoBeVisible → 削除（Liveボタンもなくなる）
- TC5: 「ログアウト時の編集が原文トグルで消えている」→ 全テスト削除
- TC6: 「ログイン時の編集が原文トグルでも消えていない」→ 全テスト削除
- TC7: 「ゲストが削除しても原文(Draft)に影響しない」→ 全テスト削除
- TC8: 「認証ユーザーが削除するとDraftからも消える」→ 全テスト削除
- TC1のDraft/Liveボタン期待値を削除

## 検証
```
cd web && go build ./... && go test ./...
cd web/cockpit && npx tsc --noEmit
cd web/cockpit && npx playwright test e2e/auth-dual-view.spec.ts
```
