package domain

import "context"

//go:generate go tool mockgen -destination=./mocks/dispatcher_mock.go -package=mocks . Dispatcher

// Dispatcher はサーバー層からアプリケーション層へのイベント配送を担当します。
type Dispatcher interface {
	// Dispatch は通常のデータイベントを配送します。
	Dispatch(ctx context.Context, data []byte) error
}

type Sender interface {
	Send(ctx context.Context, data []byte) error
}
