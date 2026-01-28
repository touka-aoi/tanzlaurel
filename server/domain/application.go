package domain

import "context"

type Application interface {
	// HandleMessage はメッセージのパースと処理を一気貫通で実行する。
	// パース結果をApplication外部に漏らさない設計。
	HandleMessage(ctx context.Context, sessionID SessionID, data []byte) error
	Tick(ctx context.Context) interface{}
}
