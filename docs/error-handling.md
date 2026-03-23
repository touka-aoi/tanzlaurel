# TanzError - エラーハンドリングユーティリティ

## 概要

`TanzError`は、構造化ログ（`log/slog`）との統合を前提としたエラーラッピングユーティリティです。エラー発生時のコンテキスト情報（メッセージ、エラーコード、任意のフィールド）を保持し、`slog`で出力可能な形式に変換します。

## パッケージ

`withered/utils/error`（パッケージ名: `TanzError`）

## 構造体

### TanzError

```go
type TanzError struct {
    Message string
    Code    string
    Fields  map[string]interface{}
    Cause   error
}
```

| フィールド | 説明 |
|-----------|------|
| `Message` | 人間が読むためのエラーメッセージ |
| `Code` | エラーを一意に識別するコード（例: `"SESSION_NOT_FOUND"`） |
| `Fields` | slog出力時に付加する任意のキーバリュー情報 |
| `Cause` | 元のエラー（`errors.Is`/`errors.As`チェーンに対応） |

## 関数

### Wrap

```go
func Wrap(err error, message string, code string, fields Fields) *TanzError
```

エラーを`TanzError`でラップします。`err`が既に`*TanzError`の場合、`slog.Warn`で二重ラップの警告を出力します。

### AttrsFromFields

```go
func AttrsFromFields(f Fields) []slog.Attr
```

`Fields`を`[]slog.Attr`に変換します。`slog`のログ出力時に使用します。

## 使用ルール

### 一度だけラップする

エラーをハンドリング・作成する際は、**一度だけ**`Wrap`を使用してエラーをラップしてください。

```go
// 正しい使い方: エラー発生箇所で一度だけラップ
result, err := doSomething()
if err != nil {
    return TanzError.Wrap(err, "処理に失敗", "PROC_FAILED", TanzError.Fields{
        "input_id": inputID,
    })
}
```

### 二重ラップの禁止

`Wrap`の引数に既に`*TanzError`が渡された場合、`slog.Warn`で警告が出力されます。これは設計上の誤りを示すため、コードを修正してください。

```go
// 禁止: 既にラップされたエラーを再度ラップ
tanzErr := TanzError.Wrap(err, "first", "CODE_1", nil)
doubled := TanzError.Wrap(tanzErr, "second", "CODE_2", nil) // slog.Warn が出力される
```

### slogでの出力例

```go
tanzErr := TanzError.Wrap(err, "セッションが見つからない", "SESSION_NOT_FOUND", TanzError.Fields{
    "session_id": sessionID,
})

attrs := TanzError.AttrsFromFields(tanzErr.Fields)
slog.LogAttrs(ctx, slog.LevelError, tanzErr.Error(), attrs...)
```

## 標準errorsパッケージとの互換性

`TanzError`は`Unwrap()`メソッドを実装しているため、`errors.Is`および`errors.As`で元のエラーチェーンをたどることができます。

```go
var sentinel = errors.New("not found")
wrapped := TanzError.Wrap(sentinel, "msg", "CODE", nil)

errors.Is(wrapped, sentinel)       // true
errors.As(wrapped, &tanzErr)       // true
```