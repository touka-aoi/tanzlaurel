# InputApplication実装計画

## 目的
InputPayload (KeyMask) をパースし、受信した入力をエコーバックするApplication実装を作成する。

## 前提（実装済み）

- `server/domain/protocol.go` - Header, PayloadHeader, InputPayload等
- `server/domain/protocol_test.go` - テストコード
- `server/domain/application.go` - Applicationインターフェース

## 実装タスク

### 1. InputApplication (server/application/input_application.go)

```go
package application

import (
    "context"
    "github.com/touka-aoi/paralle-vs-single/server/domain"
)

type InputEvent struct {
    SessionID domain.SessionID
    KeyMask   uint32
}

type InputApplication struct {
    pendingInputs []InputEvent
}

func NewInputApplication() *InputApplication

// Parse: バイト列 → InputEvent
// Header(13) + PayloadHeader(2) + InputPayload(4) = 19バイト
func (app *InputApplication) Parse(ctx context.Context, data []byte) (interface{}, error)

// Handle: 受信した入力をpendingInputsに追加
func (app *InputApplication) Handle(ctx context.Context, event interface{}) error

// Tick: pendingInputsをエンコードして返却 → ブロードキャスト
func (app *InputApplication) Tick(ctx context.Context) interface{}
```

### 2. テスト (server/application/input_application_test.go)

- Parse: 正常系・異常系
- Handle: 入力追加
- Tick: エンコード・クリア

## ファイル一覧

| ファイル | 操作 |
|---------|------|
| `server/application/input_application.go` | 新規作成 |
| `server/application/input_application_test.go` | 新規作成 |

## 実装順序

1. server/application/ ディレクトリ作成
2. input_application.go 作成
3. input_application_test.go 作成
4. テスト実行

## 検証

```bash
go test ./server/application/... -v -run Input
```
