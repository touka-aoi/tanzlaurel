# ScyllaDB移行プラン (gocqlx)

## Context

現在 `adapter/jsonfile/` でJSON/JSONLファイルベースの永続化を行っている。
プランDoc（`resilient-crunching-pond.md`）の Phase N に従い、`adapter/scylladb/` を gocqlx v3 で実装し差し替える。
Repository IFは `domain/repository.go` と `application/entry_projector.go` に定義済みで、adapter層を差し替えるだけで移行できる。

## CQLスキーマ

```cql
CREATE KEYSPACE IF NOT EXISTS flourish
  WITH replication = {'class': 'SimpleStrategy', 'replication_factor': 1};

-- イベントソース (正規データ)
CREATE TABLE IF NOT EXISTS flourish.events (
    entry_id   UUID,
    server_seq BIGINT,
    request_id UUID,
    event_type TEXT,
    site_id    UUID,
    payload    BLOB,
    created_at TIMESTAMP,
    PRIMARY KEY (entry_id, server_seq)
) WITH CLUSTERING ORDER BY (server_seq ASC);

-- L2冪等性 (TTL 7日で自動削除)
CREATE TABLE IF NOT EXISTS flourish.request_dedup (
    request_id UUID PRIMARY KEY,
    entry_id   UUID,
    created_at TIMESTAMP
) WITH default_time_to_live = 604800;

-- エントリビュー (Read Model)
CREATE TABLE IF NOT EXISTS flourish.entries (
    id         UUID PRIMARY KEY,
    title      TEXT,
    content    TEXT,
    thumbnail  TEXT,
    text       TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    deleted    BOOLEAN
);

-- RGAスナップショット (復元用)
CREATE TABLE IF NOT EXISTS flourish.rga_states (
    entry_id   UUID PRIMARY KEY,
    server_seq BIGINT,
    state      BLOB,
    updated_at TIMESTAMP
);
```

## 新規ファイル

```
server/adapter/scylladb/
├── session.go          # NewSession(hosts) → gocqlx.Session
├── schema.go           # CQL定数 + Migrate(session)
├── models.go           # row構造体 + gocqlx table定義
├── convert.go          # gocql.UUID ↔ uuid.UUID, domain型変換
├── event_store.go      # EventStore実装 (L1/L2 dedup, seq管理)
├── entry_store.go      # EntryStore実装
├── rga_state_store.go  # RGAStateStore実装
├── event_store_test.go # integration test (//go:build integration)
├── entry_store_test.go
└── rga_state_store_test.go
docker-compose.yml      # ScyllaDB単独
```

## 変更ファイル

| ファイル | 変更内容 |
|---------|---------|
| `go.mod` | `go get github.com/scylladb/gocqlx/v3` |
| `server/cmd/main.go` | jsonfile → scylladb に差し替え、SCYLLA_HOST環境変数追加 |
| `Makefile` | `db` / `db-stop` ターゲット追加 |

## 実装ステップ

### 1. docker-compose.yml 作成
- `scylladb/scylla:6.2`, ポート9042, `--smp 1 --memory 512M --developer-mode 1`

### 2. gocqlx依存追加
- `cd web && go get github.com/scylladb/gocqlx/v3`

### 3. session.go + schema.go
- `NewSession(hosts...)` → cluster設定、keyspace=flourish
- `Migrate(session)` → CREATE TABLE IF NOT EXISTS を順次実行

### 4. models.go + convert.go
- eventRow, dedupRow, entryRow, rgaStateRow 構造体 (`db:` タグ付き)
- gocqlx `table.New()` でテーブルメタデータ定義
- `toGocqlUUID` / `toUUID` / `toEntry` / `toEntryRow` / `toEventSlice` 変換関数

### 5. entry_store.go
- `Save` → `InsertQuery().BindStruct()`
- `FindByID` → `GetQuery()`, not found → `ErrEntryNotFound`, deleted → `ErrEntryDeleted`
- `List` → `SELECT id,title,content,thumbnail,created_at,updated_at,deleted FROM entries` (textを除外)
- `Delete` → FindByID → deleted=true → Save

### 6. rga_state_store.go
- `SaveRGA` → `json.Marshal(snap)` → InsertQuery
- `LoadRGA` → GetQuery → `json.Unmarshal`
- `ListRGAEntryIDs` → `SELECT entry_id FROM rga_states`

### 7. event_store.go (最も複雑)
- **initSeqs()**: `SELECT DISTINCT entry_id FROM events` → 各entryの `MAX(server_seq)` をメモリに復元
- **Append()**: L1チェック(メモリmap) → L2チェック(`INSERT IF NOT EXISTS` on request_dedup) → seq採番 → events INSERT → L1追加
- **ListAfter()**: `WHERE entry_id = ? AND server_seq > ?`
- **MaxServerSeq()**: メモリから返す
- **EntryIDs()**: メモリのseqsマップのキー一覧（インターフェース外の具象メソッド、cmd/main.goで使用）

### 8. cmd/main.go 更新
```go
scyllaHost := envOrDefault("SCYLLA_HOST", "localhost:9042")
session, err := scylladb.NewSession(scyllaHost)
// ...
scylladb.Migrate(session.Session)
entryStore := scylladb.NewEntryStore(session)
eventStore, _ := scylladb.NewEventStore(session)
rgaStateStore := scylladb.NewRGAStateStore(session)
```
- `PrintBanner` の第3引数を `scyllaHost` に変更（シグネチャは既に対応済み）

### 9. Makefile更新
```makefile
db:
	docker compose up -d scylladb
db-stop:
	docker compose down
```

### 10. integrationテスト
- `//go:build integration` タグで分離
- `testSession(t)` ヘルパー: localhost:9042に接続、不可ならSkip
- jsonfileテストと同等のケースをカバー

## 検証

1. `docker compose up -d scylladb` → 9042ポートで起動確認
2. `go run ./server/cmd/` → スキーマ自動作成、バナーにScyllaアドレス表示
3. `cd cockpit && npm run dev` → エントリ作成・編集・一覧取得が動作
4. サーバー再起動 → `initSeqs()` でserver_seq復元、既存データが保持される
5. `go test -tags integration ./server/adapter/scylladb/...`
