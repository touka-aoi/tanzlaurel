package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/service"
	appstate "github.com/touka-aoi/paralle-vs-single/application/state"
	parallelstate "github.com/touka-aoi/paralle-vs-single/server/parallel/internal/state"
	transporthttp "github.com/touka-aoi/paralle-vs-single/server/parallel/internal/transport/http"
)

func main() {
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()

	interactionState := buildState()
	metrics := &noopMetrics{}
	clock := realClock{}
	validator := service.SimpleValidator{}

	svc, err := service.NewInteractionService(interactionState, metrics, clock, validator)
	if err != nil {
		log.Fatalf("failed to construct service: %v", err)
	}

	mux := transporthttp.NewMux(transporthttp.Dependencies{
		Service: svc,
		State:   interactionState,
	})

	server := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}

	go func() {
		log.Printf("parallel single listening on %s", *addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http single error: %v", err)
		}
	}()

	waitForShutdown(server)
}

func buildState() appstate.InteractionState {
	cfg := parallelstate.Config{
		Players: seedPlayers(),
		Rooms:   seedRooms(),
	}
	return parallelstate.New(cfg)
}

func seedPlayers() []domain.PlayerSnapshot {
	// Placeholder seed data; replace with config-driven loader if needed.
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

func waitForShutdown(server *http.Server) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if err := server.Close(); err != nil {
			log.Printf("forced close failed: %v", err)
		}
	}
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

func (realClock) Since(t time.Time) time.Duration {
	return time.Since(t)
}

type noopMetrics struct{}

func (noopMetrics) RecordLatency(ctx context.Context, endpoint string, duration time.Duration) {}

func (noopMetrics) RecordContention(ctx context.Context, endpoint string, wait time.Duration) {}

func (noopMetrics) IncrementCounter(ctx context.Context, name string, delta int) {}

// Ensure noopMetrics satisfies the interface.
var _ appstate.MetricsRecorder = (*noopMetrics)(nil)
