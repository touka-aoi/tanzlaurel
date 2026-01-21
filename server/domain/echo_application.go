package domain

import "context"

// EchoApplication は受信したメッセージをそのままブロードキャストするテスト用Application。
type EchoApplication struct {
	pendingData []byte
}

func NewEchoApplication() *EchoApplication {
	return &EchoApplication{}
}

func (e *EchoApplication) Parse(ctx context.Context, data []byte) (interface{}, error) {
	return data, nil
}

func (e *EchoApplication) Handle(ctx context.Context, event interface{}) error {
	e.pendingData = event.([]byte)
	return nil
}

func (e *EchoApplication) Tick(ctx context.Context) interface{} {
	if e.pendingData == nil {
		return nil
	}
	data := e.pendingData
	e.pendingData = nil
	return data
}
