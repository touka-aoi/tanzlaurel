# クライアントサイドルーティング: 個別記事ページ

## Context

フィードから個別記事ページ（`/entries/:id`）へ遷移できるようにしたい。
URLを共有可能にするため、History APIベースのクライアントサイドルーティングを導入する。
ルーターには`preact-iso`を使用する（公式推奨、非同期ルーティング・lazy対応）。

## 変更対象ファイル

| ファイル | 変更内容 |https://github.com/preactjs/preact-iso?ref=voluntas.ghost.io
|---|---|
| `src/app.tsx` | `<LocationProvider>` + `<Router>`でフィードと個別ページを切り替え |
| `src/pages/entry-page.tsx` | **新規** 個別記事ページ（全文表示 + 編集） |
| `src/components/feed-card.tsx` | 個別ページへのリンクボタン追加 |

## 実装ステップ

### Step 1: entry-page.tsx 新規作成
- `/entries/:id` に対応するページコンポーネント
- `useDocument(id)` で CRDT 接続、全文表示
- クリックで編集モード（既存feed-cardと同じUX）
- 戻るリンク（`/` へ遷移）

### Step 1.5: preact-iso インストール
- `npm install preact-iso` で導入
- 既存の `preact-router` は `npm uninstall preact-router` で削除

### Step 2: app.tsx にルーティング追加
- `<LocationProvider>` でアプリ全体をラップ
- `<Router>` で `/` と `/entries/:id` を切り替え
- ナビゲーションは `useLocation()` の `navigate` を使用
- フィード部分はそのまま、個別ページは `EntryPage` へ

### Step 3: feed-card.tsx にリンク追加
- カード内に個別ページへ遷移する `<a href="/entries/:id">` リンクを追加（`LocationProvider` が自動でSPAナビゲーション化）
- フェードグラデーションの下あたりに配置

## 検証
```bash
cd web/cockpit && npx vite build
```
1. フィードからカードのリンクで `/entries/:id` に遷移できる
2. 個別ページで全文が表示される
3. ブラウザの戻るボタンでフィードに戻れる
4. `/entries/:id` を直接開いても表示される
