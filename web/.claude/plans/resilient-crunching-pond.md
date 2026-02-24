# CRDTブログサービス 実装プラン

## Context

既存のシューティングゲームプロジェクト (`server/`, `client/`) とは別に、`web/` ディレクトリにCRDTベースの協調編集ブログサービスを新規構築する。CRDTは学習目的で自前実装し、rapid (PBT) で正しさを数学的に検証する。

**開発手法: テスト駆動開発 (TDD)**
- 各フェーズで必ずテストを先に書き、テストが失敗することを確認してから実装する
- Red → Green → Refactor のサイクルを厳守
- CRDTコアは PBT (rapid) でプロパティを先に定義し、実装で満たす

## 技術スタック

| レイヤー | 技術 |
|---------|------|
| Frontend | Preact + Vite + TailwindCSS v4 |
| Lint | oxlint (TS), golangci-lint (Go) |
| Backend | Go 1.26, Clean Architecture |
| DB | ScyllaDB (Event Store) |
| CRDT | 自前RGA実装 (Go + TS) |
| 同期 | WebSocket (coder/websocket) |
| テスト | rapid (PBT), vitest, Playwright |

## プロジェクト構造

```
web/
├── server/
│   ├── cmd/main.go
│   ├── domain/
│   │   ├── crdt/          # RGA, LamportClock, Operation
│   │   ├── entry.go       # BlogEntryエンティティ
│   │   ├── event.go       # イベント型
│   │   └── repository.go  # リポジトリIF
│   ├── application/       # EntryService, SyncService
│   ├── adapter/
│   │   ├── scylladb/      # EventStore, EntryStore, RGAStateStore
│   │   └── websocket/     # WSトランスポート
│   ├── handler/           # HTTP/WSハンドラ
│   ├── router.go
│   └── server.go
├── cockpit/
│   └── src/
│       ├── crdt/          # RGA (TS), Clock, Operation
│       ├── sync/          # WebSocket, OfflineQueue, SyncManager
│       ├── hooks/         # useDocument, useSync, useEntries
│       ├── components/    # EntryList, EntryEditor, MarkdownPreview
│       └── lib/           # markdown-itラッパー
├── docker-compose.yml     # ScyllaDB
├── Makefile
└── .pre-commit-config.yaml
```

## CRDT設計: RGA (Replicated Growable Array)

- **NodeID**: SiteID (UUID) + Lamport Timestamp で一意識別
- **request_id**: 各オペレーションのユニークID (UUID v7)。冪等性の担保に使用
- **Node**: ID, Value(rune), Deleted(tombstone), After(挿入位置の直前ノード)
- **Operation**: Insert / Delete の2種。各opにrequest_idを付与
- **並行挿入の決定ルール**: 同位置なら Timestamp大 → 左側。同Timestampなら SiteID辞書順小 → 左側
- Go/TS両方で同一アルゴリズムを実装
- **RGA永続化**: RGA適用後の状態をシリアライズしてScyllaDBに非同期保存

## OSOTとRGAの責務分離

```
Client → op(UID付き) → Server(OSOT) → 重複検知 → server_seq採番 → ScyllaDB永続化 → Broadcast
                                                                         ↓
                                                              受信側Client: RGAがop適用時に並行解決
```

**OSOT (サーバー、単一インスタンス)** の責務:
- opの到着順管理と配信順序の決定 (server_seq採番)
- UID による重複検知・破棄 (冪等性担保)
- 全イベントの正規ソース (One Source of Truth)
- ScyllaDBにイベントを永続化 (OSOT保存先 = ScyllaDB events テーブル)

**server_seq採番方針**:
- 採番はGoメモリ上のカウンタで行う (entry_idごとにインクリメント)
- ScyllaDBは保存のみ、採番ロジックは持たない
- サーバー再起動時: ScyllaDBから `MAX(server_seq)` を取得してカウンタを復元

**RGA (各レプリカ)** の責務:
- op適用時に並行挿入の解決 (同じパイプライン内で処理)
- 最終的な文字列順序の決定

**冪等性**: 各opにユニークID (request_id) を付与。2層で重複検知:
- L1: Goメモリセット (高速判定、再起動で消える)
- L2: ScyllaDB `INSERT ... IF NOT EXISTS` (永続的担保)

**接続フロー**:

初回接続・再接続ともに同じフロー。クライアントは `sync_request` を送信し、サーバーは `sync` で応答する。

1. クライアントが `sync_request` を送信 (初回: `last_server_seq: 0`、再接続: 最後に受信したseq)
2. サーバーが `sync` で差分opを返却 (初回: 全op、再接続: 差分)
3. クライアントは `sync` に自分の未確認opが含まれているか検査
4. 含まれていなければ再送する

## DB設計

```
op到着 → events追記（同期）→ メモリRGA適用
                                  ↓ (非同期 debounce)
                             rga_states更新（state + server_seq）
                             entries更新（textからtitle, content, thumbnail導出）
```

| テーブル | 役割 |
|---|---|
| events | opの履歴（イベントソース、正規データ） |
| rga_states | RGAの最新状態（復元用、内部テーブル） |
| entries | 記事のビュー（API用、Read Model） |

### events

| カラム | 型 | 説明 |
|---|---|---|
| entry_id (PK) | UUID | エントリ識別子 |
| server_seq (CK ASC) | int64 | サーバー採番 |
| request_id | UUID v7 | 冪等性キー |
| event_type | string | `crdt_op` / `entry_create` / `entry_delete` |
| site_id | UUID | 送信元クライアント |
| payload | bytes | JSON-encoded Operation |
| created_at | datetime | |

- 差分同期: `WHERE entry_id = ? AND server_seq > ?`
- 重複検知: 2層で担保
  - L1: Goメモリ上のセット (ホットパス用キャッシュ、再起動で消える)
  - L2: ScyllaDB `INSERT ... IF NOT EXISTS` (永続化、再起動後も有効)
- ホットパーティション: ブログ記事サイズなら問題なし

### rga_states

| カラム | 型 | 説明 |
|---|---|---|
| entry_id (PK) | UUID | エントリ識別子 |
| server_seq | int64 | この状態が含むイベントの最大seq |
| state | bytes | RGAのシリアライズ（ノード構造全体） |
| updated_at | datetime | |

- サーバー再起動時: ここから復元 + 差分events再生
- APIからは直接読まない内部テーブル

### entries

| カラム | 型 | 説明 |
|---|---|---|
| id (PK) | UUID | エントリ識別子 |
| title | string | textの1行目から導出 |
| content | string | textの先頭N文字 |
| thumbnail | string? | text中の最初の画像URL |
| text | string | RGA.Text()の全文 |
| created_at | datetime | |
| updated_at | datetime | |
| deleted | bool | 論理削除フラグ |

- 一覧取得時はtextを除外してSELECT

## ログ設計

標準ライブラリ `log/slog` を使用。JSON出力。

**キー名の変更**

| デフォルト | 変更後 |
|---|---|
| time | timestamp |
| msg | message |

**共通Attr** (全ログに付与):

| キー | 説明 | 例 |
|---|---|---|
| service.name | サービス名 | "crdt-blog" |
| service.version | バージョン | "0.1.0" |
| deployment.environment | 環境 | "development" / "production" |

**拡張Attr (HTTP)**:

| キー | 型 | 説明 |
|---|---|---|
| http.request.method | string | GET, POST, DELETE 等 |
| http.response.status_code | int | レスポンスステータスコード |
| url.path | string | リクエストパス |
| user_agent.original | string | User-Agentヘッダ |
| client.address | string | クライアントIPアドレス |
| http.request.body.size | int | リクエストボディサイズ (bytes) |
| http.server.request.duration | float | リクエスト処理時間 (秒) |

**拡張Attr (DB)**:

| キー | 型 | 説明 |
|---|---|---|
| db.system | string | "scylladb" |
| db.statement | string | 実行したCQLクエリ |
| db.operation | string | "SELECT", "INSERT" 等 |
| db.duration | float | クエリ実行時間 (秒) |

**拡張Attr (エラー)**:

| キー | 型 | 説明 |
|---|---|---|
| error.type | string | エラー種別URI (`error:entry_not_found` 等) |
| error.request_id | string (UUID v7) | 原因リクエストのID |
| error.title | string | エラー種別の要約 |
| error.instance | string | この発生のID (`urn:uuid:<uuid>`) |
| error.description | string | エラーの詳細説明 (内部ログ用) |

**source**: WARN以上で発生ファイル名と行番号を出力

**Attrキー**: camelCase (`entryId`, `serverSeq`, `requestId` 等)

**ログレベル**

slogのカスタムレベルで6段階を定義。

| レベル | slog値 | 説明 |
|---|---|---|
| FATAL | 12 | 【サービス停止】アプリケーション継続不可能。必須の環境変数未設定等。os.Exit |
| ERROR | 8 | 【要対応】リクエスト処理は失敗したがアプリは継続可能。予期せぬ例外等 |
| WARN | 4 | 【潜在的問題】直ちにエラーではないが将来対応が必要。証明書期限切れ接近等 |
| INFO | 0 | 【動作記録】正常性判断のための重要イベント。バッチ開始/終了等 |
| DEBUG | -4 | 【開発者向け】バグ修正用の詳細情報。実行されたクエリ等 |
| TRACE | -8 | 【詳細追跡】DEBUGより更に詳細。メソッドの開始/終了等 |

**ログレベル制御**

環境変数 `LOG_LEVEL` で制御。デフォルトは `info`。指定レベル以上を出力。

**起動時ログ**:

`fmt.Sprint` で出力。見やすさ重視（slog JSONではない）。

```
==================================================
 Service:   crdt-blog
 Ver:       v0.1.0
 GitHash:   da94541
 Timestamp: 2025-01-01T00:00:00+09:00
 ENV:       development
 LogLevel:  info
 Address:   :8080
 Scylla:    localhost:9042
==================================================
```

バナー出力後、各サービスの起動・読み込み成功をINFOで出力:

```
{"timestamp":"...","level":"INFO","message":"scylla connected","host":"localhost:9042"}
{"timestamp":"...","level":"INFO","message":"event store initialized","entries":3}
{"timestamp":"...","level":"INFO","message":"rga states loaded","entries":3}
{"timestamp":"...","level":"INFO","message":"http server listening","address":":8080"}
```

**Web APIログ**:

リクエスト・レスポンスの両方をHTTPミドルウェアで出力する。

リクエスト (INFO):
```json
{
  "timestamp": "...",
  "level": "INFO",
  "message": "request received",
  "http.request.method": "POST",
  "url.path": "/api/entries",
  "client.address": "192.168.1.1",
  "user_agent.original": "Mozilla/5.0 ...",
  "http.request.body.size": 0,
  "request_id": "019abc12-..."
}
```

レスポンス (INFO):
```json
{
  "timestamp": "...",
  "level": "INFO",
  "message": "response sent",
  "http.request.method": "POST",
  "url.path": "/api/entries",
  "http.response.status_code": 201,
  "http.server.request.duration": 0.0032,
  "request_id": "019abc12-..."
}
```

## WebSocketプロトコル (JSON)

全メッセージは `type` フィールドで識別する。
クライアント→サーバーの全リクエストに `request_id` (UUID v7) を付与する。サーバーはrequest_idで重複検知し、受付済みなら破棄する。これによりopだけでなく、ビュー生成など他の処理の冪等性も担保する。

### `op` (クライアント→サーバー)

CRDTオペレーションを送信する。サーバーは重複検知後、永続化してACKを返す。

| フィールド | 型 | 説明 |
|---|---|---|
| type | string | 常に "op" |
| request_id | string (UUID v7) | リクエストのユニークID (冪等性) |
| entry_id | string (UUID) | 対象エントリ |
| op_type | int | 1: Insert, 2: Delete |
| node_id | NodeID | Insert: 新ノードID / Delete: 対象ノードID |
| after | NodeID? | Insert: 挿入位置の直前ノード (null=先頭) |
| value | string | Insert: 1文字 / Delete: 未使用 |

**NodeID**

| フィールド | 型 | 説明 |
|---|---|---|
| site_id | string (UUID) | クライアント識別子 |
| timestamp | int | Lamport timestamp |

```json
{
  "type": "op",
  "entry_id": "550e8400-...",
  "request_id": "019abc12-...",
  "op_type": 1,
  "node_id": { "site_id": "client-uuid", "timestamp": 42 },
  "after": { "site_id": "other-uuid", "timestamp": 10 },
  "value": "a"
}
```

### `ack` (サーバー→送信元クライアント)

リクエストの受付確認。クライアントはACK受信でキューからリクエストを安全に削除する。

| フィールド | 型 | 説明 |
|---|---|---|
| type | string | 常に "ack" |
| request_id | string (UUID v7) | 受付したリクエストのID |
| entry_id | string (UUID) | 対象エントリ |
| server_seq | int64 | サーバーが採番した順序番号 |

```json
{
  "type": "ack",
  "request_id": "019abc12-...",
  "entry_id": "550e8400-...",
  "server_seq": 1234
}
```

### `sync_request` (クライアント→サーバー)

再接続時に差分を要求する。

| フィールド | 型 | 説明 |
|---|---|---|
| type | string | 常に "sync_request" |
| request_id | string (UUID v7) | リクエストのユニークID (冪等性) |
| entry_id | string (UUID) | 対象エントリ |
| last_server_seq | int64 | クライアントが最後に受信したserver_seq |

```json
{
  "type": "sync_request",
  "request_id": "019abc14-...",
  "entry_id": "550e8400-...",
  "last_server_seq": 1200
}
```

### `sync` (サーバー→クライアント)

opをクライアントに配信する。リアルタイム配信と再接続時の差分同期を統一したメッセージ。
クライアントは `ops` を順にRGAに適用するだけでよい。

| フィールド | 型 | 説明 |
|---|---|---|
| type | string | 常に "sync" |
| entry_id | string (UUID) | 対象エントリ |
| ops | SyncOp[] | op一覧 (リアルタイム時は1件、再接続時はN件) |
| latest_server_seq | int64 | 現在のサーバー最新seq |

**SyncOp**

| フィールド | 型 | 説明 |
|---|---|---|
| request_id | string (UUID v7) | 元リクエストのID |
| server_seq | int64 | サーバー順序番号 |
| op_type | int | 1: Insert, 2: Delete |
| node_id | NodeID | ノードID |
| after | NodeID? | 挿入位置 |
| value | string | 文字 |

```json
{
  "type": "sync",
  "entry_id": "550e8400-...",
  "ops": [
    {
      "request_id": "019abc12-...",
      "server_seq": 1234,
      "op_type": 1,
      "node_id": { "site_id": "client-uuid", "timestamp": 42 },
      "after": { "site_id": "other-uuid", "timestamp": 10 },
      "value": "a"
    }
  ],
  "latest_server_seq": 1234
}
```

### `error` (サーバー→クライアント)

エラー通知。RFC 9457 (Problem Details) を参考にした構造。

| フィールド | 型 | 説明 |
|---|---|---|
| type | string | 常に "error" |
| request_id | string (UUID v7)? | エラーの原因となったリクエストのID (不明な場合はnull) |
| error_type | string (URI) | エラー種別を識別するURI (デフォルト: "about:blank") |
| title | string | エラー種別の短い要約 (同じerror_typeでは固定) |
| instance | string | この発生を一意に識別するID (urn:uuid:<uuid>) |

**error_type 一覧**

| error_type | title | 説明 |
|---|---|---|
| error:invalid_op | Invalid Operation | opのフォーマット不正、未知のentry_id等 |
| error:entry_not_found | Entry Not Found | 指定されたentry_idが存在しない |
| error:entry_deleted | Entry Deleted | 削除済みエントリへの操作 |
| error:internal | Internal Error | サーバー内部エラー |

```json
{
  "type": "error",
  "request_id": "019abc12-...",
  "error_type": "error:entry_not_found",
  "title": "Entry Not Found",
  "instance": "urn:uuid:019abc15-7def-7000-8000-000000000001"
}
```

## REST API

### `GET /api/health`

サーバーの生存確認。

**Response 200**

| フィールド | 型 | 説明 |
|---|---|---|
| status | string | 常に "ok" |

```json
{ "status": "ok" }
```

### `GET /api/entries`

エントリ一覧を取得する。

**Response 200**

| フィールド | 型 | 説明 |
|---|---|---|
| entries | EntryListItem[] | エントリ一覧 |

**EntryListItem**

| フィールド | 型 | 説明 |
|---|---|---|
| id | string (UUID) | エントリ識別子 |
| title | string | タイトル |
| content | string | 本文の先頭N文字 |
| thumbnail | string? | 本文中の最初の画像URL (なければnull) |
| created_at | string | ISO 8601 |
| updated_at | string | ISO 8601 |

```json
{
  "entries": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "はじめての投稿",
      "content": "本文の先頭N文字...",
      "thumbnail": "https://example.com/image.png",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T00:00:00Z"
    }
  ]
}
```

### `POST /api/entries`

エントリを新規作成する。リクエストボディなし。サーバーがUUIDを採番し空エントリを返す。

**Response 201**

**EntryCreated**

| フィールド | 型 | 説明 |
|---|---|---|
| id | string (UUID) | 採番されたエントリ識別子 |
| title | string | 空文字列 |
| created_at | string | ISO 8601 |
| updated_at | string | ISO 8601 |

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "title": "",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-01-01T00:00:00Z"
}
```

### `DELETE /api/entries/:id`

エントリを論理削除する。

**Response 204 (No Content)**

### エラーレスポンス (RFC 9457)

全エンドポイント共通。Content-Type: `application/problem+json`。

| フィールド | 型 | 説明 |
|---|---|---|
| type | string (URI) | エラー種別URI。HTTPステータス以上の情報がない場合は "about:blank" |
| status | int | HTTPステータスコード |
| title | string | type が "about:blank" の場合はHTTPステータス名、それ以外はエラー種別の要約 |
| instance | string | この発生を一意に識別するID (urn:uuid:<uuid>) |

**エラー種別一覧**

| status | type | title | 対象 |
|---|---|---|---|
| 404 | error:entry_not_found | Entry Not Found | GET/DELETE 存在しないentry |
| 410 | error:entry_deleted | Entry Deleted | GET/DELETE 削除済みentry |
| 500 | about:blank | Internal Server Error | HTTPステータス以上の情報なし |

```json
// ドメイン固有のエラー
{
  "type": "error:entry_not_found",
  "status": 404,
  "title": "Entry Not Found",
  "instance": "urn:uuid:019abc15-7def-7000-8000-000000000001"
}

// HTTPステータス以上の情報がない場合
{
  "type": "about:blank",
  "status": 500,
  "title": "Internal Server Error",
  "instance": "urn:uuid:019abc15-7def-7000-8000-000000000002"
}
```

## フロントエンド

- **markdown-it**: Markdownレンダリング

## 実装フェーズ

### Phase 1: CRDT コアロジック (TDD)

| Server (Go) | Client (TS) |
|---|---|
| RGA (メモリ) | — |

1. Go: テスト先行 — PBT (収束性, 可換性, 冪等性) を定義
2. Go: LamportClock, Operation型(request_id付き), RGA (Insert/Delete/Apply/Text) 実装

### Phase 2: サーバーインフラ + OSOT (TDD, インメモリ)

| Server (Go) | Client (TS) |
|---|---|
| RGA + OSOT (インメモリモック) | — |

Repository IFを定義し、インメモリ実装でOSOTの動作を先に確認する。ScyllaDB導入は後。

1. go.mod 初期化
2. domain: Entry, Event(request_id付き), Repository IF → テスト先行
3. adapter/memory: インメモリ EventStore (server_seq採番, 重複検知), EntryStore
4. handler: Health, Entry REST API, WebSocket accept
5. cmd/main.go, router.go, server.go

### Phase 3: リアルタイム同期 + RGA永続化 (TDD, インメモリ)

| Server (Go) | Client (TS) |
|---|---|
| op受信→重複検知→永続化→broadcast (インメモリ) | RGA (メモリ) + op送受信 (オンラインのみ) |
| RGA永続化 (インメモリ) | サーバーからRGA状態で復元 |

OSOTがopの塊を保持しているため、RGA状態の永続化は非同期で実行できる。
リロード時はインメモリのRGA状態 + 差分opで復元する (サーバー再起動で消える前提)。

1. TS: vitestテストを先行定義 → RGA実装 (Go版と同一アルゴリズム)
2. SyncService テスト先行 (op受信→重複検知→永続化→broadcast→ACK)
3. サーバー側CRDTドキュメント管理 (メモリRGA)
4. インメモリ RGAStateStore実装 (RGA適用後の状態を非同期シリアライズ)
5. リロード時: クライアントにRGA状態+差分opを送信するエンドポイント
6. 再接続時: sync_request/sync フロー + 自分のopが未確認なら再送
7. 同期プロトコル統合テスト

### Phase N (後日): ScyllaDB導入

| Server (Go) | Client (TS) |
|---|---|
| インメモリ → ScyllaDB に差し替え | 変更なし |

1. docker-compose.yml (ScyllaDB)
2. adapter/scylladb: EventStore, EntryStore, RGAStateStore (Repository IF実装)
3. サーバー再起動時: MAX(server_seq) 復元, RGA状態 + 差分op再生
4. 統合テスト

### Phase 4: フロントエンド

| Server (Go) | Client (TS) |
|---|---|
| — | RGA (メモリ) + UI + オフライン読み取り専用 |

1. Vite + Preact + TailwindCSS初期化
2. WebSocket接続, SyncManager (オンライン時のみ編集可)
3. オフライン時は読み取り専用 (最後のText()を表示)
4. リロード時はサーバーからRGA状態+差分で復元
5. Preact hooks (useDocument, useSync, useEntries)
6. UI: EntryList, EntryEditor, MarkdownPreview
7. Playwrightスモークテスト
8. pre-commit設定 (golangci-lint, oxlint)
9. デザインはglass-uiスキルを使用する。テーマカラーは青

### Phase 5: オフライン編集対応 (将来)

| Server (Go) | Client (TS) |
|---|---|
| — | IndexedDBにRGA状態を保持 → オフラインでも編集可能 |

1. IndexedDBにRGA状態を永続化 (クライアント側)
2. オフライン時も編集可能に (ローカルRGAに適用、opをキューに蓄積)
3. オンライン復帰時にキューのopをサーバーへ送信
4. オフライン統合テスト

## 検証方法

1. `make test` — Go PBT + ユニットテスト (RGA収束性検証)
2. `npx vitest` — TS RGAテスト
3. 2タブでブラウザを開き同時編集 → テキスト収束を目視確認
4. `make e2e` — Playwrightスモークテスト
