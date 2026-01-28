package domain

import (
	"context"
	"log/slog"
)

// EchoApplication は受信したメッセージをそのままブロードキャストするテスト用Application。
type EchoApplication struct {
	pendingData []byte
}

func NewEchoApplication() *EchoApplication {
	return &EchoApplication{}
}

func (e *EchoApplication) HandleMessage(ctx context.Context, sessionID SessionID, data []byte) error {
	e.pendingData = data
	return nil
}

func (e *EchoApplication) Tick(ctx context.Context) interface{} {
	if e.pendingData == nil {
		return nil
	}
	data := e.pendingData
	e.pendingData = nil
	slog.DebugContext(ctx, "EchoApplication Tick", "data", string(data))
	return data
}
