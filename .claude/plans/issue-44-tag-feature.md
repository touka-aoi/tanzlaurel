# Issue #44: タグ機能を追加する

## 概要
ブログ記事にタグを付与し、タグによる絞り込みができるようにする。

## 目的
記事が増えた際にカテゴリ別に整理・閲覧できるようにする。

## ゴール
- 記事にタグを付与できる
- タグ一覧の表示
- タグによる記事の絞り込み

## API
### 新規/変更エンドポイント
- `GET /api/entries?tag=xxx` — タグで絞り込んだ記事一覧
- `GET /api/tags` — 全タグ一覧（記事数付き）
- `PUT /api/admin/entries/{id}/tags` — 記事のタグを更新
  - リクエスト: `{ "tags": ["go", "crdt"] }`
  - 認証必須

### サーバー側の変更
- `web/server/domain/entry.go`: Entry に `Tags []string` フィールド追加
- `web/server/handler/entry.go`: タグ関連ハンドラ追加、ListEntries にタグフィルタ追加
- `web/server/domain/repository.go`: EntryStore に `ListByTag(tag string)` 追加

## DB
### テーブル設計（#40 DB実装後の場合）
```sql
CREATE TABLE entry_tags (
    entry_id TEXT NOT NULL REFERENCES entries(id),
    tag TEXT NOT NULL,
    PRIMARY KEY (entry_id, tag)
);

CREATE INDEX idx_entry_tags_tag ON entry_tags(tag);
```

### ファイルベースの場合
- Entry構造体にTagsフィールドを追加し、`entries.json` に含める

## テスト
- タグ付与APIの単体テスト
- タグフィルタの単体テスト
- タグ一覧APIの単体テスト
- フロントエンドでタグ表示・フィルタのE2Eテスト
