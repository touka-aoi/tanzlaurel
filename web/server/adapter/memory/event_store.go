package memory

import (
	"context"
	"sync"
	"time"

	"flourish/server/domain"

	"github.com/google/uuid"
)

// EventStore はイベントのインメモリ実装。
type EventStore struct {
	mu     sync.RWMutex
	events map[uuid.UUID][]domain.Event // entryID -> events
	seqs   map[uuid.UUID]int64         // entryID -> 最新seq
	seen   map[uuid.UUID]struct{}      // request_id -> 重複検知
}

func NewEventStore() *EventStore {
	return &EventStore{
		events: make(map[uuid.UUID][]domain.Event),
		seqs:   make(map[uuid.UUID]int64),
		seen:   make(map[uuid.UUID]struct{}),
	}
}

func (s *EventStore) Append(_ context.Context, event domain.Event) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.seen[event.RequestID]; exists {
		return 0, nil
	}
	s.seen[event.RequestID] = struct{}{}

	s.seqs[event.EntryID]++
	seq := s.seqs[event.EntryID]
	event.ServerSeq = seq
	event.CreatedAt = time.Now().UTC()
	s.events[event.EntryID] = append(s.events[event.EntryID], event)

	return seq, nil
}

func (s *EventStore) ListAfter(_ context.Context, entryID uuid.UUID, afterSeq int64) ([]domain.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[entryID]
	var result []domain.Event
	for _, e := range events {
		if e.ServerSeq > afterSeq {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *EventStore) MaxServerSeq(_ context.Context, entryID uuid.UUID) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.seqs[entryID], nil
}
