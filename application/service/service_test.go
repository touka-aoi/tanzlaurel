package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/request"
)

func TestNewInteractionService_RequiresDependencies(t *testing.T) {
	_, err := NewInteractionService(nil, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected error when dependencies are nil")
	}
}

func TestInteractionService_MoveSuccess(t *testing.T) {
	state := &mockState{
		moveResult: domain.MoveResult{
			Player: domain.PlayerSnapshot{PlayerID: "player-1"},
		},
	}
	metrics := &mockMetrics{}
	clock := &fakeClock{
		now:   time.Unix(100, 0),
		since: 15 * time.Millisecond,
	}
	validator := &mockValidator{}

	svc, err := NewInteractionService(state, metrics, clock, validator)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	ctx := context.Background()
	cmd := domain.MoveCommand{
		ActorID:      "player-1",
		RoomID:       "room-1",
		NextPosition: domain.Vec2{X: 1, Y: 2},
		Facing:       0.5,
	}
	result, err := svc.Move(ctx, request.Move{Command: cmd})
	if err != nil {
		t.Fatalf("move returned error: %v", err)
	}

	if !validator.moveCalled {
		t.Fatalf("expected validator.Move to be called")
	}
	if state.moveCmd != cmd {
		t.Fatalf("expected state to receive cmd=%+v, got %+v", cmd, state.moveCmd)
	}
	if result.Player.PlayerID != "player-1" {
		t.Fatalf("unexpected result: %+v", result.Player)
	}
	metrics.assertLatency(t, "move", 15*time.Millisecond)
	metrics.assertCounter(t, "requests.move", 1)
}

func TestInteractionService_MoveValidationError(t *testing.T) {
	state := &mockState{}
	metrics := &mockMetrics{}
	clock := &fakeClock{since: time.Millisecond}
	validator := &mockValidator{moveErr: errors.New("validation failed")}

	svc, err := NewInteractionService(state, metrics, clock, validator)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	_, err = svc.Move(context.Background(), request.Move{Command: domain.MoveCommand{ActorID: "p"}})
	if err == nil || !errors.Is(err, ErrInvalidPayload) {
		t.Fatalf("expected ErrInvalidPayload, got %v", err)
	}
	if state.moveCalled {
		t.Fatalf("state.ApplyMove should not be called on validation error")
	}
	metrics.assertLatency(t, "move", time.Millisecond)
	metrics.assertCounter(t, "requests.move", 1)
}

func TestInteractionService_MoveStateError(t *testing.T) {
	state := &mockState{
		moveErr: errors.New("state failure"),
	}
	metrics := &mockMetrics{}
	clock := &fakeClock{since: 2 * time.Millisecond}
	validator := &mockValidator{}

	svc, err := NewInteractionService(state, metrics, clock, validator)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	_, err = svc.Move(context.Background(), request.Move{Command: domain.MoveCommand{ActorID: "p", RoomID: "r"}})
	if err == nil || !errors.Is(err, state.moveErr) {
		t.Fatalf("expected state error, got %v", err)
	}
	if !state.moveCalled {
		t.Fatalf("expected state.ApplyMove to be called")
	}
	metrics.assertLatency(t, "move", 2*time.Millisecond)
	metrics.assertCounter(t, "requests.move", 1)
}

func TestInteractionService_BuffAttackTradeInvokeState(t *testing.T) {
	state := &mockState{}
	metrics := &mockMetrics{}
	clock := &fakeClock{since: time.Millisecond}
	validator := &mockValidator{}
	svc, err := NewInteractionService(state, metrics, clock, validator)
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	ctx := context.Background()
	buff := request.Buff{Command: domain.BuffCommand{CasterID: "c", RoomID: "r"}}
	if _, err := svc.Buff(ctx, buff); err != nil {
		t.Fatalf("buff error: %v", err)
	}
	if !state.buffCalled {
		t.Fatalf("expected ApplyBuff to be called")
	}

	attack := request.Attack{Command: domain.AttackCommand{AttackerID: "a", TargetID: "t", RoomID: "r"}}
	if _, err := svc.Attack(ctx, attack); err != nil {
		t.Fatalf("attack error: %v", err)
	}
	if !state.attackCalled {
		t.Fatalf("expected ApplyAttack to be called")
	}

	trade := request.Trade{Command: domain.TradeCommand{InitiatorID: "i", PartnerID: "p", RoomID: "r", Offer: []domain.ItemChange{{ItemID: "item", QuantityDelta: 1}}}}
	if _, err := svc.Trade(ctx, trade); err != nil {
		t.Fatalf("trade error: %v", err)
	}
	if !state.tradeCalled {
		t.Fatalf("expected ApplyTrade to be called")
	}
}

// --- test doubles ---

type fakeClock struct {
	now   time.Time
	since time.Duration
}

func (f *fakeClock) Now() time.Time {
	if f.now.IsZero() {
		return time.Unix(0, 0)
	}
	return f.now
}

func (f *fakeClock) Since(time.Time) time.Duration {
	return f.since
}

type mockValidator struct {
	moveCalled   bool
	buffCalled   bool
	attackCalled bool
	tradeCalled  bool

	moveErr   error
	buffErr   error
	attackErr error
	tradeErr  error
}

func (m *mockValidator) Move(request.Move) error {
	m.moveCalled = true
	return m.moveErr
}

func (m *mockValidator) Buff(request.Buff) error {
	m.buffCalled = true
	return m.buffErr
}

func (m *mockValidator) Attack(request.Attack) error {
	m.attackCalled = true
	return m.attackErr
}

func (m *mockValidator) Trade(request.Trade) error {
	m.tradeCalled = true
	return m.tradeErr
}

type mockState struct {
	moveCalled   bool
	buffCalled   bool
	attackCalled bool
	tradeCalled  bool

	moveCmd domain.MoveCommand
	buffCmd domain.BuffCommand
	attackCmd domain.AttackCommand
	tradeCmd domain.TradeCommand

	moveResult  domain.MoveResult
	buffResult  domain.BuffResult
	attackResult domain.AttackResult
	tradeResult domain.TradeResult

	moveErr  error
	buffErr  error
	attackErr error
	tradeErr error
}

func (m *mockState) ApplyMove(ctx context.Context, cmd domain.MoveCommand) (domain.MoveResult, error) {
	m.moveCalled = true
	m.moveCmd = cmd
	return m.moveResult, m.moveErr
}

func (m *mockState) ApplyBuff(ctx context.Context, cmd domain.BuffCommand) (domain.BuffResult, error) {
	m.buffCalled = true
	m.buffCmd = cmd
	return m.buffResult, m.buffErr
}

func (m *mockState) ApplyAttack(ctx context.Context, cmd domain.AttackCommand) (domain.AttackResult, error) {
	m.attackCalled = true
	m.attackCmd = cmd
	return m.attackResult, m.attackErr
}

func (m *mockState) ApplyTrade(ctx context.Context, cmd domain.TradeCommand) (domain.TradeResult, error) {
	m.tradeCalled = true
	m.tradeCmd = cmd
	return m.tradeResult, m.tradeErr
}

type mockMetrics struct {
	latencyCalls []latencyCall
	counters     map[string]int
}

type latencyCall struct {
	endpoint string
	duration time.Duration
}

func (m *mockMetrics) RecordLatency(ctx context.Context, endpoint string, duration time.Duration) {
	m.latencyCalls = append(m.latencyCalls, latencyCall{endpoint: endpoint, duration: duration})
}

func (m *mockMetrics) RecordContention(ctx context.Context, endpoint string, wait time.Duration) {
	// contention metrics are not exercised in current tests
}

func (m *mockMetrics) IncrementCounter(ctx context.Context, name string, delta int) {
	if m.counters == nil {
		m.counters = make(map[string]int)
	}
	m.counters[name] += delta
}

func (m *mockMetrics) assertLatency(t *testing.T, endpoint string, expect time.Duration) {
	t.Helper()
	if len(m.latencyCalls) == 0 {
		t.Fatalf("expected latency call for %s", endpoint)
	}
	call := m.latencyCalls[len(m.latencyCalls)-1]
	if call.endpoint != endpoint || call.duration != expect {
		t.Fatalf("unexpected latency call: %+v expected duration %v", call, expect)
	}
}

func (m *mockMetrics) assertCounter(t *testing.T, name string, expect int) {
	t.Helper()
	got := m.counters[name]
	if got != expect {
		t.Fatalf("unexpected counter %s: got %d expect %d", name, got, expect)
	}
}
