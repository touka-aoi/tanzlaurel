package domain

import (
	"context"
)

//go:generate go tool mockgen -destination=./mocks/transport_mock.go -package=mocks . Transport

// Transport は Conn（物理接続）が依存するI/O境界です。
type Transport interface {
	Read(ctx context.Context) (data []byte, err error)
	Write(ctx context.Context, data []byte) error
	Close(code int32, reason string) error
}
