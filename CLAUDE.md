# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

WebSocketを使ったリアルタイム通信システムのGoプロジェクトです。接続管理アーキテクチャに焦点を当て、スケーラブルなWebSocket処理のためのper-connection管理パターンを探求しています。

## コミュニケーションガイドライン

- **日本語で返答してください**
- コードの意思決定はdocs/adr以下にまとめています
- 一度に全てのコードを書くのではなく、関数、構造体単位でユーザーに確認をとる
- レビューガイドラインは以下に記載
  - .github/copilot-instructions.md

## 開発コマンド

### モック生成
```bash
make generate-mock
# または直接
go generate ./...
```

### テスト実行
```bash
go test ./...
# 特定のパッケージ
go test ./server/domain/...
# 詳細出力付き
go test -v ./...
```

### サーバー起動
```bash
go run server/cmd/main.go
```

環境変数:
- `ADDR`: サーバーアドレス (デフォルト: "localhost")
- `PORT`: サーバーポート (デフォルト: "9090")

## アーキテクチャ

### レイヤー構造

クリーンアーキテクチャパターンに従い、関心事が明確に分離されています:

```
server/
├── domain/       # コアドメインロジックとインターフェース
├── adapter/      # 外部アダプター (websocket transport)
├── handler/      # HTTPハンドラー
├── cmd/          # アプリケーションエントリーポイント
└── server.go     # サーバー実装
```

### 核となる概念

アーキテクチャは**物理的な接続**と**論理的な接続**を区別しています:

- **Connection (物理的な接続)**: WebSocketの物理接続を表現
  - `Transport`インターフェースでI/O操作をラップ
  - 一度切断されると再利用不可
  - `server/domain/connection.go`に実装

- **Session (論理的な接続)**: アプリケーションから見た論理接続を表現
  - アクティビティタイムスタンプを追跡 (read, write, pong)
  - アイドル検知とタイムアウト管理を担当
  - 将来的な再接続/rebindサポートを想定した設計
  - `server/domain/session.go`に実装

- **SessionEndpoint**: SessionとConnectionを橋渡しし、per-connection goroutine管理を実装
  - **owner loopパターン**を実装 (ADR-002)
  - 接続ごとに4つのgoroutineを実行:
    - `ownerLoop`: 集約された状態管理と意思決定
    - `readLoop`: ブロッキングI/O読み込み処理
    - `writeLoop`: I/O書き込み処理
    - Ticker: アイドル検知のための定期的なtickを送信
  - `server/domain/session_endpoint.go`に実装

### 主要な設計パターン

#### Per-Connection Owner Loop (ADR-002)
各接続は専用のgoroutineで管理され、以下を実現:
- owner loopで状態更新を集約し、race conditionを設計レベルで排除
- I/O処理と状態管理の関心事を分離
- 将来的な再接続シナリオに対応可能

メリット:
- 接続状態の単一の情報源
- 責務分離が明確
- session rebindへの拡張性

トレードオフ:
- 接続ごとのgoroutine数が増加

#### ルームベースのメッセージ配送 (ADR-006)
- Roomはセッショングループを管理しブロードキャスト
- 配送は**best-effort** (配送保証なし)
- バックプレッシャー処理: チャネルが満杯時に`ErrRoomBusy`を返す
- 容量管理: `Dispatcher`が満杯時に登録を拒否
- 空室が一定時間続いたルームは破棄される

### ドメインインターフェース

**Transport**: 接続の物理的なI/O境界
```go
type Transport interface {
    Read(ctx context.Context) ([]byte, error)
    Write(ctx context.Context, data []byte) error
    Close(code int32, reason string) error
}
```

**Dispatcher**: サーバー層からアプリケーション層へのイベント配送
```go
type Dispatcher interface {
    Dispatch(ctx context.Context, sessionID SessionID, data []byte) error
    DispatchControl(ctx context.Context, sessionID SessionID, event interface{}) error
    RemoveSession(sessionID SessionID)
}
```

**Application**: アプリケーションロジックのインターフェース (Roomに注入)
```go
type Application interface {
    Parse(ctx context.Context, data []byte) (interface{}, error)
    Handle(ctx context.Context, event interface{}) error
}
```

### Roomイベントループ

Roomはtickベースのイベントループ (デフォルト: 60 FPS) で以下を処理:
1. 制御イベント (セッションの追加/削除)
2. 受信イベント (受信メッセージ)
3. 送信イベント (前tickからのブロードキャスト/ユニキャスト)

これにより送信メッセージに**1フレームの遅延**が生じますが、一貫したタイミングのため意図的な設計です。

## 重要な注意事項

### エラーハンドリングの考慮事項
- `ErrBackpressure`: writeチャネルが満杯。アプリケーション層でのリトライロジックが必要
- `ErrRoomBusy`: Roomの制御/送信チャネルが満杯。バックオフ戦略が必要
- RoomのParse/Handleのエラー処理は未完成 (TODOコメントあり)

### モック生成
プロジェクトは`go.uber.org/mock`を使用:
- モック対象: `Transport`, `Dispatcher`
- モックファイルは`server/domain/mocks/`に配置
- インターフェース変更後は`make generate-mock`で再生成

### ADR参照
設計根拠の詳細は`docs/adr/`のArchitecture Decision Recordsを参照:
- ADR-001: Heartbeat管理 (Deprecated - ADR-002に置き換え)
- ADR-002: Per-connection owner loop設計
- ADR-003: Heartbeat loopの削除 (空ファイル)
- ADR-006: Roomライフサイクルと配送方針
