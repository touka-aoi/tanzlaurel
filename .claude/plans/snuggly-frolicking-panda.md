# URLペースト時のタイトルリンク変換

## Context
エディタ(entry-page)でURLをペーストしたとき、Notionのように `[ページタイトル](URL)` のMarkdownリンクに自動変換する機能を追加する。

## 変更対象

### 1. バックエンド: OGPタイトル取得API
**新規ファイル**: `web/server/handler/ogp.go`

- `GET /api/ogp?url=<encoded-url>` エンドポイントを追加
- 指定URLをサーバーからfetchし、`<title>` タグまたは `og:title` メタタグからタイトルを抽出
- レスポンス: `{ "title": "ページタイトル", "url": "https://..." }`
- タイムアウト: 5秒
- URLバリデーション（httpスキームのみ許可）

**変更ファイル**: `web/server/router.go`
- `GET /api/ogp` ルートを追加

### 2. フロントエンド: ペーストハンドラ
**変更ファイル**: `web/cockpit/src/pages/entry-page.tsx`

- textareaに `onPaste` イベントハンドラを追加
- クリップボードの内容がURL単体かどうかを判定（`https?://` で始まりスペースなし）
- URL単体の場合:
  1. `e.preventDefault()` でデフォルトのペーストを止める
  2. まず `[読み込み中...](URL)` をカーソル位置に挿入（即座にフィードバック）
  3. `/api/ogp?url=...` を呼んでタイトルを取得
  4. 取得成功: `[タイトル](URL)` に置換
  5. 取得失敗: `[URL](URL)` にフォールバック

## 検証方法
1. `cd web && go test ./server/handler/...` でハンドラのテストを実行
2. サーバー起動 → `curl "http://localhost:8080/api/ogp?url=https://example.com"` でAPI確認
3. フロント起動 → エディタにURL（例: `https://github.com`）をペーストし、タイトルリンクに変換されることを確認
