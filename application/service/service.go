package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/request"
	"github.com/touka-aoi/paralle-vs-single/application/state"
)

var (
	ErrInvalidPayload = errors.New("service: invalid payload")
)

// InteractionService はエンドポイント単位のユースケースをまとめる。
type InteractionService struct {
	state    state.InteractionState
	metrics  state.MetricsRecorder
	clock    Clock
	validate Validator
}

// NewInteractionService は依存を注入してサービスを生成する。
func NewInteractionService(s state.InteractionState, m state.MetricsRecorder, clock Clock, validator Validator) (*InteractionService, error) {
	if s == nil || m == nil || clock == nil || validator == nil {
		return nil, fmt.Errorf("service: missing dependencies: state=%v metrics=%v clock=%v validator=%v", s, m, clock, validator)
	}
	return &InteractionService{
		state:    s,
		metrics:  m,
		clock:    clock,
		validate: validator,
	}, nil
}

// Move は単一プレイヤーの移動処理を実行する。
func (s *InteractionService) Move(ctx context.Context, payload request.Move) (domain.MoveResult, error) {
	start := s.clock.Now()
	defer s.record("move", start)

	if err := s.validate.Move(payload); err != nil {
		return domain.MoveResult{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyMove(ctx, payload.Command)
}

// Buff は同一 Room 内の複数プレイヤーへバフを適用する。
func (s *InteractionService) Buff(ctx context.Context, payload request.Buff) (domain.BuffResult, error) {
	start := s.clock.Now()
	defer s.record("buff", start)

	if err := s.validate.Buff(payload); err != nil {
		return domain.BuffResult{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyBuff(ctx, payload.Command)
}

// Attack はプレイヤー間の攻撃処理を行う。
func (s *InteractionService) Attack(ctx context.Context, payload request.Attack) (domain.AttackResult, error) {
	start := s.clock.Now()
	defer s.record("attack", start)

	if err := s.validate.Attack(payload); err != nil {
		return domain.AttackResult{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyAttack(ctx, payload.Command)
}

// Trade はトレード処理を実施する。
func (s *InteractionService) Trade(ctx context.Context, payload request.Trade) (domain.TradeResult, error) {
	start := s.clock.Now()
	defer s.record("trade", start)

	if err := s.validate.Trade(payload); err != nil {
		return domain.TradeResult{}, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyTrade(ctx, payload.Command)
}

func (s *InteractionService) record(endpoint string, started time.Time) {
	duration := s.clock.Since(started)
	// 計測時は新たな context を生成してメトリクス専用に利用する
	ctx := context.Background()
	s.metrics.RecordLatency(ctx, endpoint, duration)
	s.metrics.IncrementCounter(ctx, "requests."+endpoint, 1)
}

// Clock はテスト容易性のための時間抽象化。
type Clock interface {
	Now() time.Time
	Since(time.Time) time.Duration
}

// Validator は入力検証の抽象化。
type Validator interface {
	Move(request.Move) error
	Buff(request.Buff) error
	Attack(request.Attack) error
	Trade(request.Trade) error
}
