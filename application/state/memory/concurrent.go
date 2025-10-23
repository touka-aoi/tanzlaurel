package memory

import (
	"context"
	"sync"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/state"
)

// ConcurrentStore は Store をラップし、排他制御付きで InteractionState を実装する。
type ConcurrentStore struct {
	base *Store
	clk  func() time.Time
	mu   sync.RWMutex
}

// NewConcurrentStore は新しい ConcurrentStore を生成する。
func NewConcurrentStore(base *Store) *ConcurrentStore {
	return &ConcurrentStore{
		base: base,
		clk:  time.Now,
	}
}

// WithClock はテスト用に時間ソースを差し替える。
func (c *ConcurrentStore) WithClock(clock func() time.Time) *ConcurrentStore {
	if clock != nil {
		c.clk = clock
	}
	return c
}

func (c *ConcurrentStore) ApplyMove(ctx context.Context, cmd domain.MoveCommand) (domain.MoveResult, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base.applyMove(cmd, c.now())
}

func (c *ConcurrentStore) ApplyBuff(ctx context.Context, cmd domain.BuffCommand) (domain.BuffResult, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base.applyBuff(cmd, c.now())
}

func (c *ConcurrentStore) ApplyAttack(ctx context.Context, cmd domain.AttackCommand) (domain.AttackResult, error) {
	_ = ctx
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.base.applyAttack(cmd, c.now())
}

func (c *ConcurrentStore) ApplyTrade(ctx context.Context, cmd domain.TradeCommand) (domain.TradeResult, error) {
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
