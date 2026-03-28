# HttpOnly Cookie認証への移行

## Context
現在のアクセストークンは `globalToken` (インメモリ変数) に保存されており、ページリロードで消失する。
アクセストークン・リフレッシュトークンの両方をHttpOnly cookieに移行し、JSから直接トークンを扱わない構成にする。

## 方針
- アクセストークン（短命5分）→ HttpOnly cookie `access_token`
- リフレッシュトークン（長命7日）→ HttpOnly cookie `refresh_token`
- フロントエンドはトークンを直接扱わない。認証状態は `/api/auth/status` で確認
- WSチケット取得もcookieベースで認証

## 変更内容

### 1. サーバー: `web/server/handler/auth.go`

**Login (`POST /api/auth/login`)**
- アクセストークン(JWT, 5分)を `access_token` HttpOnly cookieにSet
- リフレッシュトークン(JWT, 7日, 別secret)を `refresh_token` HttpOnly cookieにSet
- レスポンスbodyはトークンを返さない。`{ "authenticated": true }` のみ

**新規: Refresh (`POST /api/auth/refresh`)**
- `refresh_token` cookieを検証
- 有効なら新しいアクセストークンcookieをSet
- レスポンス: `{ "authenticated": true }`

**新規: Logout (`POST /api/auth/logout`)**
- 両cookieをMaxAge=0で削除
- レスポンス: 204 No Content

**AuthMiddleware 修正**
- `Authorization: Bearer` ヘッダー → cookie `access_token` から読む
- cookie JWT を検証してcontext設定

**cookie属性**
```
HttpOnly: true
SameSite: Lax
Path: /api
Secure: ENV != "development" のとき true
```

### 2. サーバー: `web/server/cmd/main.go`
- `AuthConfig` にリフレッシュトークン用secret・Secure flag追加
- 環境変数 `REFRESH_SECRET` 追加（デフォルト: JWT_SECRETと異なる値）

### 3. サーバー: `web/server/router.go`
- `POST /api/auth/refresh` ルート追加
- `POST /api/auth/logout` ルート追加

### 4. フロントエンド: `web/cockpit/src/hooks/use-auth.ts`
- `globalToken` 変数を削除
- `login()`: POSTしてcookieが設定される。レスポンスから `authenticated` を読む
- `logout()`: `POST /api/auth/logout` を呼ぶ
- `getWSTicket()`: Authorizationヘッダー不要（cookieで自動送信）
- `getAuthHeaders()`: 不要 → 削除。呼び出し元を修正
- 認証状態の初期化: マウント時に `GET /api/auth/status` を呼んで確認
- アクセストークン期限切れ時: APIが401返したら `POST /api/auth/refresh` を試行→成功ならリトライ、失敗ならログアウト状態に

### 5. フロントエンド: `web/cockpit/src/pages/entry-page.tsx`
- `getAuthHeaders()` の呼び出しを削除（cookieで自動送信）

### 6. フロントエンド: `web/cockpit/src/pages/login-page.tsx`
- ログアウト処理を `POST /api/auth/logout` に変更

### 7. テスト更新
- `web/server/handler/handler_test.go` — cookie検証に合わせて修正
- `web/cockpit/e2e/auth-dual-view.spec.ts` — cookieベース認証に合わせてE2Eテスト更新

## 対象ファイル
- `web/server/handler/auth.go` — Login/Refresh/Logout/Middleware修正
- `web/server/cmd/main.go` — AuthConfig拡張
- `web/server/router.go` — ルート追加
- `web/cockpit/src/hooks/use-auth.ts` — cookie認証に移行
- `web/cockpit/src/pages/entry-page.tsx` — getAuthHeaders削除
- `web/cockpit/src/pages/login-page.tsx` — logout API化

## 検証
- `cd web && go test ./...`
- `cd web/cockpit && npx playwright test e2e/auth-dual-view.spec.ts`
