package TanzError

import (
	"errors"
	"log/slog"
)

type Fields map[string]any

type TanzError struct {
	Message string
	Code    string
	Fields  map[string]interface{}
	Cause   error
}

func (e TanzError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Code != "" {
		return e.Code
	}
	return "error"
}

func (e TanzError) Unwrap() error {
	return e.Cause
}

func New(message string, code string, fields Fields) *TanzError {
	return &TanzError{
		Message: message,
		Code:    code,
		Fields:  fields,
	}
}

func Wrap(err error, message string, code string, fields Fields) *TanzError {
	var existing *TanzError
	if errors.As(err, &existing) {
		slog.Warn("TanzError is already wrapped",
			slog.String("original_message", existing.Message),
			slog.String("original_code", existing.Code),
		)
	}
	return &TanzError{
		Message: message,
		Code:    code,
		Fields:  fields,
		Cause:   err,
	}
}

func AttrsFromFields(f Fields) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(f))
	for k, v := range f {
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}
