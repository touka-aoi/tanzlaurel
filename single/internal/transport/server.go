package transport

import (
	"context"
	"fmt"

	"github.com/touka-aoi/paralle-vs-single/application/request"
	"github.com/touka-aoi/paralle-vs-single/application/service"
	appstate "github.com/touka-aoi/paralle-vs-single/application/state"
	"github.com/touka-aoi/paralle-vs-single/single/internal/loop"
)

// Dependencies describes collaborators required by the single-loop transport.
type Dependencies struct {
	Service *service.InteractionService
	State   appstate.InteractionState
}

// LoopRequest represents the decoded result of the network layer.
type LoopRequest interface{}

// Supported loop request types.
type (
	MoveRequest   = request.Move
	BuffRequest   = request.Buff
	AttackRequest = request.Attack
	TradeRequest  = request.Trade
)

// LoopHandler is invoked by the event loop to process decoded requests.
type LoopHandler interface {
	Handle(req LoopRequest) error
}

type loopHandler struct {
	svc *service.InteractionService
}

func (h *loopHandler) Handle(req interface{}) error {
	ctx := context.Background()
	switch v := req.(type) {
	case MoveRequest:
		_, err := h.svc.Move(ctx, v)
		return err
	case BuffRequest:
		_, err := h.svc.Buff(ctx, v)
		return err
	case AttackRequest:
		_, err := h.svc.Attack(ctx, v)
		return err
	case TradeRequest:
		_, err := h.svc.Trade(ctx, v)
		return err
	default:
		return fmt.Errorf("transport: unsupported request type %T", req)
	}
}

// LoopRunner provides an entry point for the event loop.
type LoopRunner struct {
    Loop *loop.Loop
}

// Submit enqueues a decoded request to the loop.
func (r *LoopRunner) Submit(ctx context.Context, req LoopRequest) error {
    return r.Loop.Submit(ctx, req)
}

// NewLoopRunner constructs loop and handler wiring.
func NewLoopRunner(cfg loop.Config, deps Dependencies) (*LoopRunner, error) {
	handler := &loopHandler{svc: deps.Service}
	cfg.Handler = handler
	l, err := loop.New(cfg)
	if err != nil {
		return nil, err
	}
	return &LoopRunner{Loop: l}, nil
}
