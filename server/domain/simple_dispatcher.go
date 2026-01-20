package domain

import (
	"context"
	"log/slog"
)

// loopbackDispatcher ループバックするディスパッチャー
type loopbackDispatcher struct{}

var _ Dispatcher = (*loopbackDispatcher)(nil)

func (l loopbackDispatcher) Dispatch(ctx context.Context, data []byte) error {
	slog.DebugContext(ctx, "LoopbackDispatcher received data")
	return nil
}

func NewLoopbackDispatcher() Dispatcher {
	return &loopbackDispatcher{}
}
