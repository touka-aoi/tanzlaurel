package domain

import "context"

type Application interface {
	// TODO: 将来的には Event 型を導入して interface{} を廃止する想定。
	Parse(ctx context.Context, data []byte) (interface{}, error)
	Handle(ctx context.Context, event interface{}) error
	Tick(ctx context.Context) interface{}
}
