package memory

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"flourish/server/domain"
)

// EntryStore はエントリのインメモリ実装。
type EntryStore struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]domain.Entry
}

func NewEntryStore() *EntryStore {
	return &EntryStore{
		entries: make(map[uuid.UUID]domain.Entry),
	}
}

func (s *EntryStore) Save(_ context.Context, entry domain.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[entry.ID] = entry
	return nil
}

func (s *EntryStore) FindByID(_ context.Context, id uuid.UUID) (domain.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[id]
	if !ok {
		return domain.Entry{}, domain.ErrEntryNotFound
	}
	if entry.Deleted {
		return domain.Entry{}, domain.ErrEntryDeleted
	}
	return entry, nil
}

func (s *EntryStore) List(_ context.Context) ([]domain.EntryListItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var items []domain.EntryListItem
	for _, entry := range s.entries {
		if !entry.Deleted {
			items = append(items, entry.ToListItem())
		}
	}
	return items, nil
}

func (s *EntryStore) Delete(_ context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[id]
	if !ok {
		return domain.ErrEntryNotFound
	}
	entry.Deleted = true
	s.entries[id] = entry
	return nil
}
