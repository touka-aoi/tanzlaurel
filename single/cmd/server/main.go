package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/service"
	appstate "github.com/touka-aoi/paralle-vs-single/application/state"
	"github.com/touka-aoi/paralle-vs-single/single/internal/loop"
	singlestate "github.com/touka-aoi/paralle-vs-single/single/internal/state"
	singletransport "github.com/touka-aoi/paralle-vs-single/single/internal/transport"
)

func main() {
	addr := flag.String("addr", ":9090", "TCP listen address")
	flag.Parse()

	state := buildState()
	metrics := &noopMetrics{}
	clock := realClock{}
	validator := service.SimpleValidator{}

	svc, err := service.NewInteractionService(state, metrics, clock, validator)
	if err != nil {
		log.Fatalf("failed to construct service: %v", err)
	}

	loopRunner, err := singletransport.NewLoopRunner(loop.Config{
		QueueSize: 4096,
	}, singletransport.Dependencies{
		Service: svc,
		State:   state,
	})
	if err != nil {
		log.Fatalf("failed to build loop runner: %v", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := loopRunner.Loop.Start(ctx); err != nil {
		log.Fatalf("failed to start loop: %v", err)
	}

	go func() {
		if err := listenAndServe(ctx, *addr, loopRunner); err != nil {
			log.Printf("server error: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Println("shutdown initiated")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := loopRunner.Loop.Stop(shutdownCtx); err != nil {
		log.Printf("loop stop error: %v", err)
	}
}

func listenAndServe(ctx context.Context, addr string, runner *singletransport.LoopRunner) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	log.Printf("single-loop server listening on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		go handleConnection(ctx, conn, runner)
	}
}

func handleConnection(ctx context.Context, conn net.Conn, runner *singletransport.LoopRunner) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var frame inboundFrame
		if err := decoder.Decode(&frame); err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return
			}
			log.Printf("decode error: %v", err)
			return
		}

		req, err := frame.toLoopRequest()
		if err != nil {
			log.Printf("frame error: %v", err)
			continue
		}

		if err := runner.Submit(ctx, req); err != nil {
			log.Printf("enqueue error: %v", err)
			return
		}
	}
}

func buildState() appstate.InteractionState {
	cfg := singlestate.Config{
		Players: seedPlayers(),
		Rooms:   seedRooms(),
	}
	return singlestate.New(cfg)
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

type inboundFrame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func (f inboundFrame) toLoopRequest() (singletransport.LoopRequest, error) {
	switch f.Type {
	case "move":
		var payload singletransport.MoveRequest
		if err := json.Unmarshal(f.Payload, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case "buff":
		var payload singletransport.BuffRequest
		if err := json.Unmarshal(f.Payload, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case "attack":
		var payload singletransport.AttackRequest
		if err := json.Unmarshal(f.Payload, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	case "trade":
		var payload singletransport.TradeRequest
		if err := json.Unmarshal(f.Payload, &payload); err != nil {
			return nil, err
		}
		return payload, nil
	default:
		return nil, errors.New("unknown frame type: " + f.Type)
	}
}
