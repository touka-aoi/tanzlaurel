# Phase 2: JWT認証の導入

## Context
Basic認証ベースのログインAPI → JWT Cookie + CSRF保護を追加。WebSocketは短命チケットで認証。認証済み接続のCRDT操作にサーバーが `authenticated: true` を強制付与する。

## 認証フロー

```
[Login]
POST /api/login (Authorization: Basic xxx)
  → Set-Cookie: token=<JWT 1h> (HttpOnly, SameSite=Strict)

[API Call (state-changing)]
Cookie: token=<JWT>
  → http.CrossOriginProtection がCSRF保護
  → JWT検証ミドルウェア

[WebSocket]
POST /api/ws-ticket (Cookie)
  → Response: { ticket: "xxx" } (1分間有効、使い捨て)
GET /api/ws?ticket=xxx
  → サーバーがチケット検証 → authenticated = true/false

[Logout]
POST /api/logout
  → Cookie削除 (Max-Age=0)
```

## 要件
- 単一管理者（`ADMIN_USER` + `ADMIN_PASSWORD` env var）
- JWT署名: RS256（秘密鍵で署名、公開鍵で検証）
- JWT有効期限: 1時間
- CSRF保護: Go 1.25 `http.CrossOriginProtection` (Fetch metadataベース、トークン不要)
- WSチケット: 1分間有効、1回限り使い捨て
- `authenticated`フラグはサーバーが強制付与（クライアントの自称は上書き）

## 実装手順

### Step 1: 認証パッケージ作成
**`web/server/auth/jwt.go` (新規)**
- RS256でのJWT発行・検証（`golang-jwt/jwt/v5` 使用）
- `Sign(userID string) (string, error)` — 秘密鍵で署名
- `Verify(tokenStr string) (*Claims, error)` — 公開鍵で検証
- Claims: `sub`, `exp`
- 鍵ペア: `JWT_PRIVATE_KEY_PATH`

**`web/server/auth/ticket.go` (新規)**
- WSチケット管理（in-memory）
- `Issue() string` — ランダムトークン発行、1分TTL
- `Redeem(ticket string) bool` — 使い捨て消費、期限切れ自動削除

### Step 2: 認証ハンドラー
**`web/server/handler/auth.go` (新規)**
- `POST /api/login`: `r.BasicAuth()` でユーザー名+パスワード検証 → JWT Cookie設定
- `POST /api/logout`: Cookie Max-Age=0 で削除
- `POST /api/ws-ticket`: JWT Cookie検証 → チケット発行

### Step 3: CSRF保護 + 認証ミドルウェア
**`web/server/router.go`**
- `http.CrossOriginProtection` をstate-changing APIに適用
  - POST/DELETE /api/entries, POST /api/logout, POST /api/ws-ticket

**`web/server/handler/auth.go`**
- JWT検証ミドルウェア: Cookie → JWT検証 → `context.WithValue` で認証状態を伝搬

### Step 4: WSハンドラー認証
**`web/server/handler/ws.go`**
- `ServeHTTP()`: `r.URL.Query().Get("ticket")` でチケット取得・検証
- 認証済み → `authenticated = true`
- チケットなし/無効 → `authenticated = false`（接続は許可、非認証として扱う）
- 接続直後に `{"type":"auth_status","authenticated":bool}` 送信
- `handleOp()`: `msg.Authenticated = authenticated`（サーバーが上書き）

### Step 5: ルーター・サーバー設定
**`web/server/router.go`**
- `/api/login`, `/api/logout`, `/api/ws-ticket` ルート追加

**`web/server/cmd/main.go`**
- `ADMIN_USER`, `ADMIN_PASSWORD` env var読み取り
- RS256鍵ペア生成/読み込み

### Step 6: フロントエンド認証
**`web/cockpit/src/hooks/use-auth.ts` (新規)**
- `login(username, password)`: `POST /api/login` (Basic認証ヘッダー)
- `logout()`: `POST /api/logout`
- `getWsTicket()`: `POST /api/ws-ticket` → ticket取得
- 認証状態管理

**`web/cockpit/src/sync/ws-client.ts`**
- URL に `?ticket=xxx` 追加オプション

**`web/cockpit/src/hooks/use-document.ts`**
- WS接続前に `getWsTicket()` → チケット付きURLで接続

**`web/cockpit/src/sync/sync-manager.ts`**
- `auth_status` メッセージを受信して `authenticated` を設定

### Step 7: ログイン/ログアウトページ
**`web/cockpit/src/pages/login-page.tsx` (新規)**
- `/login` ルート: ユーザー名+パスワード入力フォーム → `POST /api/login`
- 認証成功 → フィードページへリダイレクト

**`web/cockpit/src/pages/logout-page.tsx` (新規)**
- `/logout` ルート: `POST /api/logout` 実行 → フィードページへリダイレクト

**`web/cockpit/src/app.tsx`**
- `/login`, `/logout` ルート追加

## 依存ライブラリ追加
- `github.com/golang-jwt/jwt/v5`

## 変更ファイル
- `web/server/auth/jwt.go` (新規)
- `web/server/auth/ticket.go` (新規)
- `web/server/handler/auth.go` (新規)
- `web/server/handler/ws.go`
- `web/server/router.go`
- `web/server/cmd/main.go`
- `web/cockpit/src/hooks/use-auth.ts` (新規)
- `web/cockpit/src/sync/ws-client.ts`
- `web/cockpit/src/hooks/use-document.ts`
- `web/cockpit/src/sync/sync-manager.ts`
- `web/cockpit/src/pages/login-page.tsx` (新規)
- `web/cockpit/src/pages/logout-page.tsx` (新規)
- `web/cockpit/src/app.tsx`

## セキュリティ設計
- JWT: RS256署名、HttpOnly Cookie、SameSite=Strict
- CSRF: `http.CrossOriginProtection`（Fetch metadataベース）
- WSチケット: 1分TTL、1回限り使い捨て、in-memory管理
- `authenticated`フラグ: サーバーが強制付与

## 検証

### 手動検証
1. `ADMIN_USER=admin ADMIN_PASSWORD=test go run ./server/cmd/` で起動
2. `POST /api/login` (Basic admin:test) → JWT Cookie取得
3. `POST /api/ws-ticket` → チケット取得
4. `GET /api/ws?ticket=xxx` → `auth_status: true` 受信
5. チケットなしでWS接続 → `auth_status: false`

### E2Eテスト (`web/cockpit/e2e/auth.spec.ts` 新規)
既存パターン (`auth-dual-view.spec.ts`) に倣い、Playwright E2Eテストを追加:

1. **ログインできる**: `/login` ページでユーザー名+パスワード入力 → 送信 → フィードページにリダイレクト
2. **ログアウトできる**: ログイン済み状態で `/logout` にアクセス → Cookie削除 → フィードページにリダイレクト
3. **認証済みで編集できる**: ログイン → エントリ作成 → エントリページで編集 → APIでテキスト確認

※ playwright.config.ts で既に `ADMIN_USER=admin ADMIN_PASS=pass` が設定済み
