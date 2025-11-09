package memory

import (
	"context"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/state"
)

type SingleThreadStore struct {
	base *Store
	clk  func() time.Time
}

func NewSingleThreadStore(base *Store) *SingleThreadStore {
	return &SingleThreadStore{
		base: base,
		clk:  time.Now,
	}
}

func (s *SingleThreadStore) WithClock(clock func() time.Time) *SingleThreadStore {
	if clock != nil {
		s.clk = clock
	}
	return s
}

func (s *SingleThreadStore) ApplyMove(ctx context.Context, cmd domain.MoveCommand) (domain.MoveResult, error) {
	_ = ctx
	return s.base.applyMove(cmd, s.now())
}

func (s *SingleThreadStore) ApplyBuff(ctx context.Context, cmd domain.BuffCommand) (domain.BuffResult, error) {
	_ = ctx
	return s.base.applyBuff(cmd, s.now())
}

func (s *SingleThreadStore) ApplyAttack(ctx context.Context, cmd domain.AttackCommand) (domain.AttackResult, error) {
	_ = ctx
	return s.base.applyAttack(cmd, s.now())
}

func (s *SingleThreadStore) ApplyTrade(ctx context.Context, cmd domain.TradeCommand) (domain.TradeResult, error) {
	_ = ctx
	return s.base.applyTrade(cmd, s.now())
}

func (s *SingleThreadStore) now() time.Time {
	if s.clk == nil {
		return time.Now()
	}
	return s.clk()
}

var _ state.InteractionState = (*SingleThreadStore)(nil)
