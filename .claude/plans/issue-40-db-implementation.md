# Issue #40: DB実装

## 概要
現在ファイルベース（`data/entries.json`, `data/events/{id}.jsonl`）で永続化しているブログデータをデータベースに移行する。

## 目的
- ファイルベースの永続化はスケーラビリティと信頼性に限界がある
- 同時書き込みの安全性向上
- クエリ機能の強化（タグ検索、ソートなど将来の拡張に対応）

## ゴール
- EntryStore / EventStore のDB実装を追加
- 既存のjsonfile実装を置き換え
- マイグレーション機構の導入

## DB
### テーブル設計
```sql
CREATE TABLE entries (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '',
    content TEXT NOT NULL DEFAULT '',
    text TEXT NOT NULL DEFAULT '',
    thumbnail TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    entry_id TEXT NOT NULL REFERENCES entries(id),
    server_seq INTEGER NOT NULL,
    request_id TEXT NOT NULL,
    type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entry_id, request_id)
);

CREATE TABLE rga_snapshots (
    entry_id TEXT PRIMARY KEY REFERENCES entries(id),
    snapshot BLOB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### DB選定
- SQLite（単一サーバー前提のため軽量DBで十分）
- ドライバ: `modernc.org/sqlite`（CGOフリー）

## API
- インターフェース変更なし（`domain/repository.go` の EntryStore / EventStore を満たす新実装を追加）

## インフラ
- SQLiteファイルは `data/blog.db` に配置
- マイグレーション: `web/server/adapter/sqlite/migrations/` にSQLファイルを配置
- 起動時に自動マイグレーション実行

## テスト
- SQLite実装の単体テスト（既存のjsonfileテストと同等のケース）
- 既存のjsonfileからSQLiteへのデータ移行スクリプトのテスト
- 統合テスト: サーバー起動→CRUD操作→データ永続化の確認
