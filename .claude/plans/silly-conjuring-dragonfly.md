# 認証機能の復元（巻き戻し + 再作成）

## Context

認証削除を実行したが取り消し。認証・anonymous保護・削除UIを前回と同じ設計で復元する。
変更済みファイルは会話冒頭で読んだ元の内容に書き戻す。削除した4ファイルはuntrackedだったためgitから復元不可→再作成する。

---

## Step 1: 変更済みファイルの巻き戻し（元の内容に書き戻す）

会話冒頭で読み取った元の内容をそのまま書き戻す。対象14ファイル:

### サーバー側（元の内容が会話に残っている）
1. `web/server/domain/event.go` — `Anonymous bool` フィールド復元
2. `web/server/adapter/jsonfile/event_store.go` — `eventJSON.Anonymous` + 読み書き復元
3. `web/server/application/sync_service.go` — `SyncOp.Anonymous`, `HandleOp` の `anonymous bool` 引数復元
4. `web/server/application/sync_service_test.go` — `HandleOp` に `false` 引数復元
5. `web/server/application/entry_projector.go` — `anonInserts`, `isAnonNode`, `Apply` の `anonymous bool` 復元
6. `web/server/application/entry_projector_test.go` — `Apply` に `false` 引数復元
7. `web/server/handler/ws.go` — `Auth` フィールド、チケット認証、anonymous/blocked ロジック復元
8. `web/server/handler/ws_protocol.go` — `SyncOpMsg.Anonymous` 復元
9. `web/server/handler/ws_test.go` — `NewWS` に `nil` 引数復元
10. `web/server/router.go` — `auth` 引数、認証ルート3件、ミドルウェアチェーン復元
11. `web/server/cmd/main.go` — auth初期化ブロック復元

### フロントエンド（元の内容が会話に残っている）
12. `web/cockpit/src/app.tsx` — `useAuth`, 認証ドット, LoginPageルート復元
13. `web/cockpit/src/components/feed-card.tsx` — `useAuth`, `onDelete` prop, 削除ボタン復元
14. `web/cockpit/src/sync/sync-manager.ts` — `authenticated`, `authNodeIds`, `ticketProvider`, `isAuthNode`, `nodeKey` 復元
15. `web/cockpit/src/sync/ws-client.ts` — `ticketProvider`, `setTicketProvider`, チケット付与ロジック復元
16. `web/cockpit/src/hooks/use-document.ts` — `ticketProvider` 引数復元
17. `web/cockpit/src/pages/entry-page.tsx` — `useAuth`, `getWSTicket` 条件分岐復元
18. `web/cockpit/src/pages/feed-page.tsx` — `useAuth`, `getWSTicket`, `onDelete`, `deleteEntry` 復元
19. `web/cockpit/e2e/auth-dual-view.spec.ts` — TC1〜TC4, ヘルパー復元

### Go依存
20. `web/go.mod` / `web/go.sum` — `go mod tidy` で削除されたJWT依存を `go get` で再追加

---

## Step 2: 削除ファイルの再作成（4ファイル）

これらはuntrackedだったためgit復元不可。元の設計に基づき再作成する。

### 2-1. `web/server/handler/auth.go`

インターフェース（router.go/ws.go/cmd/main.goの参照から推定）:

```go
type AuthConfig struct {
    AdminUser string
    AdminPass string
    JWTSecret []byte
}

type Auth struct { ... }

func NewAuth(config AuthConfig) *Auth
func (a *Auth) Login(w http.ResponseWriter, r *http.Request)      // POST /api/auth/login
func (a *Auth) Status(w http.ResponseWriter, r *http.Request)     // GET /api/auth/status
func (a *Auth) WSTicket(w http.ResponseWriter, r *http.Request)   // GET /api/auth/ws-ticket
func (a *Auth) AuthMiddleware(next http.Handler) http.Handler      // JWT検証ミドルウェア（コンテキストに認証状態を設定）
func (a *Auth) ValidateAndConsumeTicket(ticket string) bool        // WS用ワンタイムチケット検証
func CrossOriginProtection(next http.Handler) http.Handler         // クロスオリジン保護
```

実装内容:
- Login: admin/pass照合 → JWT発行 → `{"token": "xxx"}` レスポンス
- Status: Authorizationヘッダー検証 → `{"authenticated": true/false}`
- WSTicket: JWT検証 → ワンタイムチケット生成・保存 → `{"ticket": "xxx"}`
- AuthMiddleware: Bearer token検証、認証状態をcontextに設定（認証失敗でもリクエストは通す）
- ValidateAndConsumeTicket: チケットの存在確認+消費
- CrossOriginProtection: Origin/Refererチェック

### 2-2. `web/cockpit/src/hooks/use-auth.ts`

インターフェース（app.tsx/feed-page.tsx/entry-page.tsx/feed-card.tsxの参照から推定）:

```typescript
export function useAuth(): {
    authenticated: boolean;
    getWSTicket: () => Promise<string | null>;
}
```

実装内容:
- `globalToken: string | null` をモジュールスコープで保持
- `login(user, pass)` → POST /api/auth/login → tokenを保存
- `logout()` → tokenをクリア
- `getWSTicket()` → GET /api/auth/ws-ticket (Authorization: Bearer) → ticket返却
- `authenticated` → tokenがnon-nullかどうか

### 2-3. `web/cockpit/src/components/login-form.tsx`

- ユーザー名/パスワード入力フォーム
- ログインボタン → `useAuth().login()` を呼ぶ
- エラー表示

### 2-4. `web/cockpit/src/pages/login-page.tsx`

- LoginFormを表示
- ログイン済みならログアウトボタン表示
- ログイン成功時/ログアウト時に `/` にリダイレクト

---

## 検証

```bash
cd web && go build ./... && go test ./...
cd web/cockpit && npx tsc --noEmit
```
