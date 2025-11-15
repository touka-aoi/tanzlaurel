package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/touka-aoi/paralle-vs-single/domain"
	parallelhandler "github.com/touka-aoi/paralle-vs-single/handler/parallel"
	"github.com/touka-aoi/paralle-vs-single/repository/state"
	"github.com/touka-aoi/paralle-vs-single/repository/state/memory"
	"github.com/touka-aoi/paralle-vs-single/server/parallel"
	"github.com/touka-aoi/paralle-vs-single/service"
	"github.com/touka-aoi/paralle-vs-single/utils"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var addr string
	addr = utils.GetEnvDefault("ADDR", "localhost")
	port := utils.GetEnvDefault("PORT", "9090")

	stateRepository := buildState()
	metrics := &noopMetrics{}
	svc, err := service.NewInteractionService(stateRepository, metrics)
	if err != nil {
		log.Fatalf("failed to construct service: %v", err)
	}

	h := parallelhandler.NewHandler(svc)
	s := parallel.NewServer(fmt.Sprintf("%s:%s", addr, port), h)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http single error: %v", err)
		}
	}()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if err := s.Close(); err != nil {
			log.Printf("forced close failed: %v", err)
		}
	}
}

func buildState() state.InteractionState {
	store := memory.NewStoreWithSnapshots(seedPlayers(), seedRooms())
	return memory.NewConcurrentStore(store)
}

func seedPlayers() []domain.PlayerSnapshot {
	return []domain.PlayerSnapshot{
		{PlayerID: "player-1", RoomID: "room-1"},
		{PlayerID: "player-2", RoomID: "room-1"},
	}
}

func seedRooms() []domain.RoomSnapshot {
	return []domain.RoomSnapshot{
		{RoomID: "room-1", MemberIDs: []string{"player-1", "player-2"}},
	}
}

type noopMetrics struct{}

func (noopMetrics) RecordLatency(ctx context.Context, endpoint string, duration time.Duration) {}

func (noopMetrics) RecordContention(ctx context.Context, endpoint string, wait time.Duration) {}

func (noopMetrics) IncrementCounter(ctx context.Context, name string, delta int) {}
