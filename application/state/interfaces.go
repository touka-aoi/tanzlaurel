package state

import (
	"context"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
)

type InteractionState interface {
	ApplyMove(ctx context.Context, cmd domain.MoveCommand) (domain.MoveResult, error)
	ApplyBuff(ctx context.Context, cmd domain.BuffCommand) (domain.BuffResult, error)
	ApplyAttack(ctx context.Context, cmd domain.AttackCommand) (domain.AttackResult, error)
	ApplyTrade(ctx context.Context, cmd domain.TradeCommand) (domain.TradeResult, error)
}

type MetricsRecorder interface {
	RecordLatency(ctx context.Context, endpoint string, duration time.Duration)
	RecordContention(ctx context.Context, endpoint string, wait time.Duration)
	IncrementCounter(ctx context.Context, name string, delta int)
}
