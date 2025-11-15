package memory

import (
	"context"
	"sync"
	"time"

	"github.com/touka-aoi/paralle-vs-single/domain"
	"github.com/touka-aoi/paralle-vs-single/repository/state"
)

type ConcurrentStore struct {
	base *Store
	clk  func() time.Time
	mu   sync.RWMutex
}

func NewConcurrentStore(base *Store) *ConcurrentStore {
	return &ConcurrentStore{
		base: base,
		clk:  time.Now,
	}
}

func (c *ConcurrentStore) WithClock(clock func() time.Time) *ConcurrentStore {
	if clock != nil {
		c.clk = clock
	}
	return c
}

func (c *ConcurrentStore) ApplyMove(ctx context.Context, cmd *state.Move) (*domain.MoveResult, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base.applyMove(cmd, c.now())
}

func (c *ConcurrentStore) ApplyBuff(ctx context.Context, cmd *state.Buff) (*domain.BuffResult, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base.applyBuff(cmd, c.now())
}

func (c *ConcurrentStore) ApplyAttack(ctx context.Context, cmd *state.Attack) (*domain.AttackResult, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base.applyAttack(cmd, c.now())
}

func (c *ConcurrentStore) ApplyTrade(ctx context.Context, cmd *state.Trade) (*domain.TradeResult, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base.applyTrade(cmd, c.now())
}

func (c *ConcurrentStore) now() time.Time {
	if c.clk == nil {
		return time.Now()
	}
	return c.clk()
}

var _ state.InteractionState = (*ConcurrentStore)(nil)
