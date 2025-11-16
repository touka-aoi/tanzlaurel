package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/touka-aoi/paralle-vs-single/domain"
	"github.com/touka-aoi/paralle-vs-single/handler"
	"github.com/touka-aoi/paralle-vs-single/repository/state"
	"github.com/touka-aoi/paralle-vs-single/utils"
)

var (
	ErrInvalidPayload = errors.New("service: invalid payload")
)

type InteractionService interface {
	Connect(ctx context.Context) (string, string, error)
	Move(ctx context.Context, payload *handler.MovePayload) (*domain.MoveResult, error)
	Attack(ctx context.Context, payload *handler.AttackPayload) (*domain.AttackResult, error)
}

type interactionService struct {
	state   state.InteractionState
	metrics state.MetricsRecorder
}

func NewInteractionService(state state.InteractionState, metics state.MetricsRecorder) (InteractionService, error) {
	return &interactionService{
		state:   state,
		metrics: metics,
	}, nil
}

func (s *interactionService) Move(ctx context.Context, payload *handler.MovePayload) (*domain.MoveResult, error) {
	start := time.Now()
	defer s.record("move", start)
	if err := s.validate(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyMove(ctx, &state.Move{
		RoomID: payload.RoomID,
		MoveCommand: domain.MoveCommand{
			UserID:       payload.Command.UserID,
			NextPosition: payload.Command.NextPosition,
			Facing:       payload.Command.Facing,
		},
	})
}

func (s *interactionService) Attack(ctx context.Context, payload *handler.AttackPayload) (*domain.AttackResult, error) {
	start := time.Now()
	defer s.record("attack", start)
	if err := s.validate(payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	return s.state.ApplyAttack(ctx, &state.Attack{
		RoomID: payload.RoomID,
		AttackCommand: domain.AttackCommand{
			UserID:   payload.Command.UserID,
			TargetID: payload.Command.TargetID,
			Damage:   payload.Command.Damage,
		},
	})
}

func (s *interactionService) Connect(ctx context.Context) (string, string, error) {
	_ = ctx
	start := time.Now()
	defer s.record("connect", start)
	playerID := strings.ReplaceAll(uuid.NewString(), "-", "")
	roomID := "1"
	_ = s.state.RegisterPlayer(ctx, playerID, roomID)
	return playerID, roomID, nil
}

func (s *interactionService) record(endpoint string, started time.Time) {
	duration := time.Since(started)
	ctx := context.Background()
	s.metrics.RecordLatency(ctx, endpoint, duration)
	s.metrics.IncrementCounter(ctx, "requests."+endpoint, 1)
}

func (s *interactionService) validate(payload utils.Validator) error {
	return payload.Validate()
}
