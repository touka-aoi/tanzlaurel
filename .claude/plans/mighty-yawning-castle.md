# サーバー分割計画: server/realtime と server/web

## Context
現在 `server/cmd/main.go` が単一エントリポイントとしてREST API（ブログ/認証）とWebSocket（ゲーム/CRDT同期）の両方を`:9090`で提供している。ブログ要素とリアルタイム要素を独立してデプロイ・スケール可能にするため、2つのサーバーに分割する。

## ディレクトリ構成（変更後）

```
server/
├── domain/          # 共有（変更なし）
├── adapter/         # 共有（変更なし）
├── application/     # realtime専用（変更なし）
├── server.go        # 共有HTTPサーバーユーティリティ（変更なし）
├── web/
│   ├── handler/     # ブログ/認証ハンドラー
│   │   ├── doc_api.go   ← server/handler/doc_api.go から移動
│   │   └── auth.go      ← server/handler/auth.go から移動
│   └── cmd/
│       └── main.go  # REST APIサーバー :8080
├── realtime/
│   ├── handler/     # WebSocketハンドラー
│   │   ├── accept.go    ← server/handler/accept.go から移動
│   │   ├── doc_ws.go    ← server/handler/doc_ws.go から移動
│   │   └── health.go    ← server/handler/health.go から移動
│   └── cmd/
│       └── main.go  # WebSocketサーバー :9090
├── handler/         # 削除（各サーバーに分散）
├── cmd/main.go      # 削除
└── router.go        # 削除
```

## 実装ステップ

### Step 1: handlerを分割移動
- `server/handler/doc_api.go`, `auth.go` → `server/web/handler/`（package名: `handler`）
- `server/handler/accept.go`, `doc_ws.go`, `health.go` → `server/realtime/handler/`（package名: `handler`）
- import pathを各ファイルで更新（`withered/server/web/handler`, `withered/server/realtime/handler`）
- 旧 `server/handler/` ディレクトリ削除

### Step 2: `server/web/cmd/main.go` 作成
- PostgreSQL接続 + DocRepository + Migrate
- SessionStore + AuthHandler + DocAPI
- ルーティング: `/api/docs/*`, `/api/auth/*`, `/health`
- ポート: `8080`（ENV: `WEB_PORT`）

### Step 3: `server/realtime/cmd/main.go` 作成
- PubSub + RoomManager + Room + Application起動
- DocumentStore（CRDT同期用）
- ルーティング: `/ws`, `/ws/doc`, `/health`
- ポート: `9090`（ENV: `PORT`、既存互換）

### Step 4: 旧ファイル削除
- `server/cmd/main.go` 削除
- `server/router.go` 削除

### Step 5: Makefile更新
- `server` → `web-server`（go run server/web/cmd/main.go）+ `realtime-server`（go run server/realtime/cmd/main.go）
- `dev` で両サーバー + web + bot 起動

### Step 6: Vite proxy更新 (`web/vite.config.ts`)
- `/api` → `http://localhost:8080`
- `/ws`, `/ws/doc` → `ws://localhost:9090`（変更なし）

### Step 7: progress.md更新

## 検証
1. `make db-up` でPostgreSQL起動
2. `make dev` で全サービス起動
3. ブラウザで `/blog`, `/admin` → REST API（:8080）経由で動作確認
4. `/editor/{id}` → WebSocket（:9090）経由でCRDT同期確認
5. `/demo` → ゲームWebSocket動作確認
