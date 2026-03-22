package application

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"flourish/server/domain"
)

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
	event := domain.Event{
		EntryID:   entryID,
		RequestID: requestID,
		EventType: domain.EventCRDTOp,
		SiteID:    siteID,
		Payload:   payload,
	}

	serverSeq, err := s.eventStore.Append(ctx, event)
	if err != nil {
		return AckMessage{}, err
	}

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
	events, err := s.eventStore.ListAfter(ctx, entryID, afterSeq)
	if err != nil {
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
		return SyncMessage{}, err
	}

	return SyncMessage{
		EntryID:         entryID,
		Ops:             ops,
		LatestServerSeq: maxSeq,
	}, nil
}
