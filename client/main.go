package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/request"
)

type mode string

const (
	modeHTTP mode = "http"
	modeTCP  mode = "tcp"
)

func main() {
	var (
		modeFlag        = flag.String("mode", string(modeHTTP), "client mode: http or tcp")
		addrFlag        = flag.String("addr", "http://localhost:8080", "target address (scheme required for http)")
		requestTypeFlag = flag.String("request", "move", "request type: move|buff|attack|trade")
		totalFlag       = flag.Int("total", 100, "total number of requests to send")
		concurrencyFlag = flag.Int("concurrency", 10, "number of concurrent workers")
		actorFlag       = flag.String("actor", "player-1", "actor/player identifier")
		roomFlag        = flag.String("room", "room-1", "room identifier")
	)
	flag.Parse()

	if *totalFlag <= 0 {
		log.Fatalf("total must be positive")
	}
	if *concurrencyFlag <= 0 {
		log.Fatalf("concurrency must be positive")
	}
	if *concurrencyFlag > *totalFlag {
		*concurrencyFlag = *totalFlag
	}

	cfg := loadConfig{
		Mode:        mode(*modeFlag),
		Addr:        *addrFlag,
		RequestType: *requestTypeFlag,
		Total:       *totalFlag,
		Concurrency: *concurrencyFlag,
		ActorID:     *actorFlag,
		RoomID:      *roomFlag,
	}

	start := time.Now()
	var err error
	switch cfg.Mode {
	case modeHTTP:
		err = runHTTP(cfg)
	case modeTCP:
		err = runTCP(cfg)
	default:
		log.Fatalf("unsupported mode: %s", cfg.Mode)
	}
	if err != nil {
		log.Fatalf("load failed: %v", err)
	}
	log.Printf("completed %d %s requests in %s", cfg.Total, cfg.RequestType, time.Since(start))
}

type loadConfig struct {
	Mode        mode
	Addr        string
	RequestType string
	Total       int
	Concurrency int
	ActorID     string
	RoomID      string
}

func runHTTP(cfg loadConfig) error {
	u, err := url.Parse(cfg.Addr)
	if err != nil {
		return fmt.Errorf("invalid addr: %w", err)
	}
	path := endpointPath(cfg.RequestType)
	if path == "" {
		return fmt.Errorf("unsupported request type %s for http", cfg.RequestType)
	}
	u.Path = path

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	var success int64
	var failure int64
	var wg sync.WaitGroup
	wg.Add(cfg.Concurrency)

	requestsPerWorker, remainder := divideWork(cfg.Total, cfg.Concurrency)
	reqID := uint64(0)

	for worker := 0; worker < cfg.Concurrency; worker++ {
		count := requestsPerWorker
		if worker < remainder {
			count++
		}
		go func(workerID, n int) {
			defer wg.Done()
			for i := 0; i < n; i++ {
				id := atomic.AddUint64(&reqID, 1)
				payload, err := buildPayload(cfg, workerID, int(id))
				if err != nil {
					log.Printf("[worker %d] payload error: %v", workerID, err)
					atomic.AddInt64(&failure, 1)
					continue
				}
				data, err := json.Marshal(payload)
				if err != nil {
					log.Printf("[worker %d] marshal error: %v", workerID, err)
					atomic.AddInt64(&failure, 1)
					continue
				}
				req, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewReader(data))
				if err != nil {
					log.Printf("[worker %d] request error: %v", workerID, err)
					atomic.AddInt64(&failure, 1)
					continue
				}
				req.Header.Set("Content-Type", "application/json")
				resp, err := client.Do(req)
				if err != nil {
					log.Printf("[worker %d] http error: %v", workerID, err)
					atomic.AddInt64(&failure, 1)
					continue
				}
				_ = resp.Body.Close()
				if resp.StatusCode >= 200 && resp.StatusCode < 300 {
					atomic.AddInt64(&success, 1)
				} else {
					log.Printf("[worker %d] non-success status: %s", workerID, resp.Status)
					atomic.AddInt64(&failure, 1)
				}
			}
		}(worker, count)
	}
	wg.Wait()
	log.Printf("http mode complete (success=%d failure=%d)", success, failure)
	return nil
}

func runTCP(cfg loadConfig) error {
	var success int64
	var failure int64
	var wg sync.WaitGroup
	wg.Add(cfg.Concurrency)

	requestsPerWorker, remainder := divideWork(cfg.Total, cfg.Concurrency)
	reqID := uint64(0)

	for worker := 0; worker < cfg.Concurrency; worker++ {
		count := requestsPerWorker
		if worker < remainder {
			count++
		}
		go func(workerID, n int) {
			defer wg.Done()
			conn, err := net.Dial("tcp", cfg.Addr)
			if err != nil {
				log.Printf("[worker %d] dial error: %v", workerID, err)
				return
			}
			defer conn.Close()
			encoder := json.NewEncoder(conn)
			for i := 0; i < n; i++ {
				id := atomic.AddUint64(&reqID, 1)
				payload, err := buildPayload(cfg, workerID, int(id))
				if err != nil {
					log.Printf("[worker %d] payload error: %v", workerID, err)
					atomic.AddInt64(&failure, 1)
					continue
				}
				frame := frameFor(cfg.RequestType, payload)
				if frame == nil {
					log.Printf("[worker %d] unsupported request type %s", workerID, cfg.RequestType)
					atomic.AddInt64(&failure, 1)
					continue
				}
				if err := encoder.Encode(frame); err != nil {
					log.Printf("[worker %d] encode error: %v", workerID, err)
					atomic.AddInt64(&failure, 1)
					return
				}
				atomic.AddInt64(&success, 1)
			}
		}(worker, count)
	}
	wg.Wait()
	log.Printf("tcp mode complete (success=%d failure=%d)", success, failure)
	return nil
}

func divideWork(total, workers int) (int, int) {
	return total / workers, total % workers
}

func buildPayload(cfg loadConfig, workerID int, seq int) (interface{}, error) {
	switch cfg.RequestType {
	case "move":
		return buildMovePayload(cfg, workerID, seq), nil
	case "buff":
		return buildBuffPayload(cfg, workerID, seq), nil
	case "attack":
		return buildAttackPayload(cfg, workerID, seq), nil
	case "trade":
		return buildTradePayload(cfg, workerID, seq), nil
	default:
		return nil, fmt.Errorf("unsupported request type %s", cfg.RequestType)
	}
}

func buildMovePayload(cfg loadConfig, workerID, seq int) request.Move {
	return request.Move{
		Meta: newMeta(cfg.ActorID, workerID, seq),
		Command: domain.MoveCommand{
			ActorID:      cfg.ActorID,
			RoomID:       cfg.RoomID,
			NextPosition: domain.Vec2{X: rand.Float64() * 10, Y: rand.Float64() * 10},
			Facing:       rand.Float64() * 6.28,
		},
	}
}

func buildBuffPayload(cfg loadConfig, workerID, seq int) request.Buff {
	return request.Buff{
		Meta: newMeta(cfg.ActorID, workerID, seq),
		Command: domain.BuffCommand{
			CasterID:  cfg.ActorID,
			RoomID:    cfg.RoomID,
			TargetIDs: []string{cfg.ActorID},
			Effect: domain.BuffEffect{
				EffectID:  "atk-up",
				Magnitude: 1.25,
				Duration:  5 * time.Second,
			},
		},
	}
}

func buildAttackPayload(cfg loadConfig, workerID, seq int) request.Attack {
	return request.Attack{
		Meta: newMeta(cfg.ActorID, workerID, seq),
		Command: domain.AttackCommand{
			AttackerID:        cfg.ActorID,
			TargetID:          fmt.Sprintf("%s-target", cfg.ActorID),
			RoomID:            cfg.RoomID,
			SkillID:           "slash",
			Damage:            25,
			AdditionalEffects: []string{"bleed"},
		},
	}
}

func buildTradePayload(cfg loadConfig, workerID, seq int) request.Trade {
	return request.Trade{
		Meta: newMeta(cfg.ActorID, workerID, seq),
		Command: domain.TradeCommand{
			InitiatorID: cfg.ActorID,
			PartnerID:   fmt.Sprintf("%s-partner", cfg.ActorID),
			RoomID:      cfg.RoomID,
			Offer: []domain.ItemChange{
				{ItemID: "gold", QuantityDelta: 10},
			},
			Request: []domain.ItemChange{
				{ItemID: "potion", QuantityDelta: 1},
			},
			RequiresConfirmation: false,
		},
	}
}

func newMeta(actor string, workerID, seq int) request.Meta {
	return request.Meta{
		RequestID: fmt.Sprintf("%s-%d-%d", actor, workerID, seq),
		TraceID:   fmt.Sprintf("trace-%d-%d", workerID, seq),
		OccurredAt: time.Now(),
	}
}

func endpointPath(requestType string) string {
	switch requestType {
	case "move":
		return "/move"
	case "buff":
		return "/buff"
	case "attack":
		return "/attack"
	case "trade":
		return "/trade"
	default:
		return ""
	}
}

func frameFor(requestType string, payload interface{}) interface{} {
	return struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    requestType,
		Payload: payload,
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
