package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/repository/state"
	"github.com/touka-aoi/paralle-vs-single/handler"
)

var (
	ErrInvalidPayload = errors.New("service: invalid payload")
)

type InteractionService struct {
	state   state.InteractionState
	metrics state.MetricsRecorder
}

func NewInteractionService(state state.InteractionState, metics state.MetricsRecorder) (*InteractionService, error) {
	return &InteractionService{
		state:   state,
		metrics: metics,
	}, nil
}

func (s *InteractionService) Move(ctx context.Context, payload *handler.MovePayload) (*domain.MoveResult, error) {
	start := time.Now()
	defer s.record("move", start)
	if err := s.validate(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyMove(ctx, payload.Command)
}

func (s *InteractionService) Buff(ctx context.Context, payload *handler.BuffPayload) (*domain.BuffResult, error) {
	start := time.Now()
	defer s.record("buff", start)
	if err := s.validate(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyBuff(ctx, payload.Command)
}

func (s *InteractionService) Attack(ctx context.Context, payload *handler.AttackPayload) (*domain.AttackResult, error) {
	start := time.Now()
	defer s.record("attack", start)
	if err := s.validate(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyAttack(ctx, payload.Command)
}

func (s *InteractionService) Trade(ctx context.Context, payload *handler.TradePayload) (*domain.TradeResult, error) {
	start := time.Now()
	defer s.record("trade", start)
	if err := s.validate(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyTrade(ctx, payload.Command)
}

func (s *InteractionService) record(endpoint string, started time.Time) {
	duration := time.Since(started)
	ctx := context.Background()
	s.metrics.RecordLatency(ctx, endpoint, duration)
	s.metrics.IncrementCounter(ctx, "requests."+endpoint, 1)
}

func (s *InteractionService) validate(payload Validator) error {
	return payload.Validate()
}
