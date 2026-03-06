package jsonfile

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"flourish/server/domain"

	"github.com/google/uuid"
)

// EventStore はJSONLファイルベースのEventStore実装。
// data/events/{entryID}.jsonl にエントリごとのイベントを保存する。
type EventStore struct {
	mu     sync.RWMutex
	dir    string // data/events/
	events map[uuid.UUID][]domain.Event
	seqs   map[uuid.UUID]int64
	seen   map[uuid.UUID]struct{}
}

func NewEventStore(dataDir string) (*EventStore, error) {
	dir := filepath.Join(dataDir, "events")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create events dir: %w", err)
	}

	s := &EventStore{
		dir:    dir,
		events: make(map[uuid.UUID][]domain.Event),
		seqs:   make(map[uuid.UUID]int64),
		seen:   make(map[uuid.UUID]struct{}),
	}

	if err := s.loadAll(); err != nil {
		return nil, fmt.Errorf("load events: %w", err)
	}

	return s, nil
}

// eventJSON はJSONL保存用の構造体。
type eventJSON struct {
	EntryID   string          `json:"entry_id"`
	ServerSeq int64           `json:"server_seq"`
	RequestID string          `json:"request_id"`
	EventType string          `json:"event_type"`
	SiteID    string          `json:"site_id"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt string          `json:"created_at"`
}

func (s *EventStore) loadAll() error {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil // ディレクトリが空なら問題なし
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		entryIDStr := strings.TrimSuffix(e.Name(), ".jsonl")
		entryID, err := uuid.Parse(entryIDStr)
		if err != nil {
			continue
		}

		f, err := os.Open(filepath.Join(s.dir, e.Name()))
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB per line
		for scanner.Scan() {
			var ej eventJSON
			if err := json.Unmarshal(scanner.Bytes(), &ej); err != nil {
				continue
			}
			requestID, _ := uuid.Parse(ej.RequestID)
			siteID, _ := uuid.Parse(ej.SiteID)
			createdAt, _ := time.Parse(time.RFC3339Nano, ej.CreatedAt)

			ev := domain.Event{
				EntryID:   entryID,
				ServerSeq: ej.ServerSeq,
				RequestID: requestID,
				EventType: domain.EventType(ej.EventType),
				SiteID:    siteID,
				Payload:   []byte(ej.Payload),
				CreatedAt: createdAt,
			}
			s.events[entryID] = append(s.events[entryID], ev)
			s.seen[requestID] = struct{}{}
			if ej.ServerSeq > s.seqs[entryID] {
				s.seqs[entryID] = ej.ServerSeq
			}
		}
		f.Close()
	}

	return nil
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

	// ファイルに追記
	if err := s.appendToFile(event); err != nil {
		return 0, fmt.Errorf("append to file: %w", err)
	}

	return seq, nil
}

func (s *EventStore) appendToFile(event domain.Event) error {
	path := filepath.Join(s.dir, event.EntryID.String()+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	ej := eventJSON{
		EntryID:   event.EntryID.String(),
		ServerSeq: event.ServerSeq,
		RequestID: event.RequestID.String(),
		EventType: string(event.EventType),
		SiteID:    event.SiteID.String(),
		Payload:   event.Payload,
		CreatedAt: event.CreatedAt.Format(time.RFC3339Nano),
	}

	data, err := json.Marshal(ej)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
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

// EntryIDs は保存されている全エントリIDを返す。
func (s *EventStore) EntryIDs() []uuid.UUID {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(s.events))
	for id := range s.events {
		ids = append(ids, id)
	}
	return ids
}
