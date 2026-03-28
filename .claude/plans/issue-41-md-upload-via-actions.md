# Issue #41: GitHub Actions経由でMarkdownを直接アップロード

## 概要
Markdownファイルをリポジトリにpushするだけで、GitHub Actionsがブログに記事を自動反映する。

## 目的
- ブラウザを開かずにブログ投稿できるワークフローを実現
- Git管理されたMarkdownを正とするCMS的運用を可能にする

## ゴール
- リポジトリの特定ディレクトリ（例: `web/posts/`）にmdファイルをpush
- GitHub ActionsがAPIを叩いて記事を作成/更新
- frontmatter（title, dateなど）をサポート

## API
### 新規エンドポイント
- `POST /api/admin/entries/import` — Markdown本文を受け取り記事を作成/更新
  - リクエスト: `{ "title": "...", "text": "...", "slug": "..." }`
  - 認証: APIキーまたはCF Accessサービストークン

### サーバー側の変更
- `web/server/handler/entry.go`: importハンドラ追加
- `web/server/router.go`: ルート追加
- 記事作成時にCRDT操作をスキップし、直接テキストを保存するバルクインポートパス

## インフラ
### GitHub Actions ワークフロー
- `.github/workflows/blog-publish.yml`
- トリガー: `web/posts/**` へのpush
- ステップ: 変更されたmdファイルを検出 → frontmatter解析 → APIにPOST
- シークレット: APIキーまたはCFサービストークン

### ディレクトリ構造
```
web/posts/
  2026-03-25-first-post.md   # frontmatter + markdown本文
```

## テスト
- importエンドポイントの単体テスト
- frontmatter解析のテスト
- GitHub Actionsワークフローのローカル検証（act等）
