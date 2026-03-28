package TanzError

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

// Message が設定されている場合、Message を返す
func TestError_WithMessage_ReturnsMessage(t *testing.T) {
	e := &TanzError{Message: "something failed", Code: "ERR_001"}
	if got := e.Error(); got != "something failed" {
		t.Errorf("Error() = %q, want %q", got, "something failed")
	}
}

// Message が空で Code が設定されている場合、Code を返す
func TestError_WithoutMessage_WithCode_ReturnsCode(t *testing.T) {
	e := &TanzError{Code: "ERR_002"}
	if got := e.Error(); got != "ERR_002" {
		t.Errorf("Error() = %q, want %q", got, "ERR_002")
	}
}

// Message も Code も空の場合、デフォルト文字列 "error" を返す
func TestError_WithoutMessageAndCode_ReturnsDefault(t *testing.T) {
	e := &TanzError{}
	if got := e.Error(); got != "error" {
		t.Errorf("Error() = %q, want %q", got, "error")
	}
}

// Cause に設定したエラーを返す
func TestUnwrap_ReturnsCause(t *testing.T) {
	cause := errors.New("root cause")
	e := &TanzError{Cause: cause}
	if got := e.Unwrap(); got != cause {
		t.Errorf("Unwrap() = %v, want %v", got, cause)
	}
}

// Cause が nil の場合、nil を返す
func TestUnwrap_NilCause_ReturnsNil(t *testing.T) {
	e := &TanzError{}
	if got := e.Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}

// Cause なしで TanzError を生成し、全フィールドが正しく設定される
func TestNew_SetsAllFields(t *testing.T) {
	fields := Fields{"key": "value"}

	got := New("new error", "NEW_001", fields)

	if got.Message != "new error" {
		t.Errorf("Message = %q, want %q", got.Message, "new error")
	}
	if got.Code != "NEW_001" {
		t.Errorf("Code = %q, want %q", got.Code, "NEW_001")
	}
	if got.Cause != nil {
		t.Errorf("Cause = %v, want nil", got.Cause)
	}
	if got.Fields["key"] != "value" {
		t.Errorf("Fields[\"key\"] = %v, want %q", got.Fields["key"], "value")
	}
}

// New で生成したエラーは error インターフェースを満たす
func TestNew_ImplementsErrorInterface(t *testing.T) {
	var err error = New("test", "CODE", nil)
	if err.Error() != "test" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test")
	}
}

// エラーを TanzError でラップし、全フィールドが正しく設定される
func TestWrap_SetsAllFields(t *testing.T) {
	cause := errors.New("original error")
	fields := Fields{"key": "value"}

	got := Wrap(cause, "wrapped message", "WRAP_001", fields)

	if got.Message != "wrapped message" {
		t.Errorf("Message = %q, want %q", got.Message, "wrapped message")
	}
	if got.Code != "WRAP_001" {
		t.Errorf("Code = %q, want %q", got.Code, "WRAP_001")
	}
	if got.Cause != cause {
		t.Errorf("Cause = %v, want %v", got.Cause, cause)
	}
	if got.Fields["key"] != "value" {
		t.Errorf("Fields[\"key\"] = %v, want %q", got.Fields["key"], "value")
	}
}

// nil エラーもラップできる
func TestWrap_NilError_Wraps(t *testing.T) {
	got := Wrap(nil, "no cause", "NIL_001", nil)

	if got.Message != "no cause" {
		t.Errorf("Message = %q, want %q", got.Message, "no cause")
	}
	if got.Cause != nil {
		t.Errorf("Cause = %v, want nil", got.Cause)
	}
}

// logRecord は slog.Handler を実装し、ログレコードをキャプチャする
type logRecord struct {
	records []slog.Record
}

func (h *logRecord) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *logRecord) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}
func (h *logRecord) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *logRecord) WithGroup(_ string) slog.Handler      { return h }

// 二重ラップ時に slog.Warn が出力される
func TestWrap_DoubleWrap_EmitsWarn(t *testing.T) {
	handler := &logRecord{}
	original := slog.Default()
	slog.SetDefault(slog.New(handler))
	t.Cleanup(func() { slog.SetDefault(original) })

	inner := Wrap(errors.New("root"), "inner message", "INNER_001", nil)
	_ = Wrap(inner, "outer message", "OUTER_001", nil)

	found := false
	for _, r := range handler.records {
		if r.Level == slog.LevelWarn && r.Message == "TanzError is already wrapped" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected slog.Warn for double wrap, but no warning was emitted")
	}
}

// errors.As で TanzError を取得できる
func TestWrap_ErrorsAs_FindsTanzError(t *testing.T) {
	cause := errors.New("root cause")
	wrapped := Wrap(cause, "wrapped", "CODE_001", Fields{"id": 42})

	var target *TanzError
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As failed to find *TanzError")
	}
	if target.Message != "wrapped" {
		t.Errorf("Message = %q, want %q", target.Message, "wrapped")
	}
}

// errors.Is で Cause のエラーを検出できる
func TestWrap_ErrorsIs_FindsCause(t *testing.T) {
	sentinel := errors.New("sentinel error")
	wrapped := Wrap(sentinel, "wrapped", "CODE_002", nil)

	if !errors.Is(wrapped, sentinel) {
		t.Error("errors.Is failed to find sentinel error through TanzError chain")
	}
}

// Fields を slog.Attr に変換する
func TestAttrsFromFields_ConvertsToSlogAttrs(t *testing.T) {
	fields := Fields{
		"user_id": 123,
		"action":  "login",
	}

	attrs := AttrsFromFields(fields)

	if len(attrs) != 2 {
		t.Fatalf("len(attrs) = %d, want 2", len(attrs))
	}

	attrMap := make(map[string]slog.Value)
	for _, a := range attrs {
		attrMap[a.Key] = a.Value
	}

	if v, ok := attrMap["user_id"]; !ok {
		t.Error("attrs does not contain key \"user_id\"")
	} else if v.String() != "123" {
		t.Errorf("user_id = %v, want 123", v)
	}

	if v, ok := attrMap["action"]; !ok {
		t.Error("attrs does not contain key \"action\"")
	} else if v.String() != "login" {
		t.Errorf("action = %v, want %q", v, "login")
	}
}

// 空の Fields の場合、空のスライスを返す
func TestAttrsFromFields_EmptyFields_ReturnsEmptySlice(t *testing.T) {
	attrs := AttrsFromFields(Fields{})

	if len(attrs) != 0 {
		t.Errorf("len(attrs) = %d, want 0", len(attrs))
	}
}
