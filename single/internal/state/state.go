package state

import (
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	appstate "github.com/touka-aoi/paralle-vs-single/application/state"
	"github.com/touka-aoi/paralle-vs-single/application/state/memory"
)

type Config struct {
	Players []domain.PlayerSnapshot
	Rooms   []domain.RoomSnapshot
	Clock   func() time.Time
}

func New(cfg Config) appstate.InteractionState {
	base := memory.NewStore(cfg.Players, cfg.Rooms)
	store := memory.NewSingleThreadStore(base)
	if cfg.Clock != nil {
		store.WithClock(cfg.Clock)
	}
	return store
}
