package application

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"flourish/server/domain"
)

var tracer = otel.Tracer("flourish/sync")

// Subscriber はsyncメッセージの受信者。
type Subscriber interface {
	Send(msg SyncMessage)
}

// SyncOp はsyncメッセージ内の個別オペレーション。
type SyncOp struct {
	RequestID uuid.UUID `json:"request_id"`
	ServerSeq int64     `json:"server_seq"`
	Payload   []byte    `json:"payload"`
}

// SyncMessage はクライアントに配信するsyncメッセージ。
type SyncMessage struct {
	EntryID         uuid.UUID `json:"entry_id"`
	Ops             []SyncOp  `json:"ops"`
	LatestServerSeq int64     `json:"latest_server_seq"`
}

// AckMessage はクライアントに返すACK。
type AckMessage struct {
	RequestID uuid.UUID `json:"request_id"`
	EntryID   uuid.UUID `json:"entry_id"`
	ServerSeq int64     `json:"server_seq"`
}

// SyncService はOSOTとしてopの受信・永続化・配信を管理する。
type SyncService struct {
	eventStore  domain.EventStore
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]Subscriber // entryID -> subscribers
}

func NewSyncService(eventStore domain.EventStore) *SyncService {
	return &SyncService{
		eventStore:  eventStore,
		subscribers: make(map[uuid.UUID][]Subscriber),
	}
}

// Subscribe はエントリのsyncメッセージを購読する。
func (s *SyncService) Subscribe(entryID uuid.UUID, sub Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subscribers[entryID] = append(s.subscribers[entryID], sub)
}

// Unsubscribe は購読を解除する。
func (s *SyncService) Unsubscribe(entryID uuid.UUID, sub Subscriber) {
	s.mu.Lock()
	defer s.mu.Unlock()
	subs := s.subscribers[entryID]
	for i, existing := range subs {
		if existing == sub {
			s.subscribers[entryID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}
}

// HandleOp はopを受信し、重複検知→永続化→ACK返却する。broadcastは別途Broadcastを呼ぶ。
func (s *SyncService) HandleOp(ctx context.Context, entryID, siteID, requestID uuid.UUID, payload []byte) (AckMessage, error) {
	ctx, span := tracer.Start(ctx, "SyncService.HandleOp",
		trace.WithAttributes(
			attribute.String("osot.entry_id", entryID.String()),
			attribute.String("osot.site_id", siteID.String()),
			attribute.String("osot.request_id", requestID.String()),
		),
	)
	defer span.End()

	event := domain.Event{
		EntryID:   entryID,
		RequestID: requestID,
		EventType: domain.EventCRDTOp,
		SiteID:    siteID,
		Payload:   payload,
	}

	serverSeq, err := s.eventStore.Append(ctx, event)
	if err != nil {
		span.RecordError(err)
		return AckMessage{}, err
	}

	span.SetAttributes(
		attribute.Int64("osot.server_seq", serverSeq),
		attribute.Bool("osot.duplicate", serverSeq == 0),
	)

	return AckMessage{
		RequestID: requestID,
		EntryID:   entryID,
		ServerSeq: serverSeq,
	}, nil
}

// Broadcast はsyncメッセージを全subscriberに配信する。
func (s *SyncService) Broadcast(entryID uuid.UUID, msg SyncMessage) {
	s.mu.RLock()
	subs := s.subscribers[entryID]
	s.mu.RUnlock()

	for _, sub := range subs {
		sub.Send(msg)
	}
}

// GetDiff は指定されたserver_seq以降の差分を取得する。
func (s *SyncService) GetDiff(ctx context.Context, entryID uuid.UUID, afterSeq int64) (SyncMessage, error) {
	ctx, span := tracer.Start(ctx, "SyncService.GetDiff",
		trace.WithAttributes(
			attribute.String("osot.entry_id", entryID.String()),
			attribute.Int64("osot.after_seq", afterSeq),
		),
	)
	defer span.End()

	events, err := s.eventStore.ListAfter(ctx, entryID, afterSeq)
	if err != nil {
		span.RecordError(err)
		return SyncMessage{}, err
	}

	ops := make([]SyncOp, len(events))
	for i, e := range events {
		ops[i] = SyncOp{
			RequestID: e.RequestID,
			ServerSeq: e.ServerSeq,
			Payload:   e.Payload,
		}
	}

	maxSeq, err := s.eventStore.MaxServerSeq(ctx, entryID)
	if err != nil {
		span.RecordError(err)
		return SyncMessage{}, err
	}

	span.SetAttributes(
		attribute.Int("osot.ops_count", len(ops)),
		attribute.Int64("osot.latest_server_seq", maxSeq),
	)

	return SyncMessage{
		EntryID:         entryID,
		Ops:             ops,
		LatestServerSeq: maxSeq,
	}, nil
}
