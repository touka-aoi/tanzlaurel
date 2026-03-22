package jsonfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"flourish/server/domain"

	"github.com/google/uuid"
)

// EntryStore はJSONファイルベースのEntryStore実装。
// data/entries.json に全エントリを保存する。
type EntryStore struct {
	mu      sync.RWMutex
	path    string
	entries map[uuid.UUID]domain.Entry
}

// entryJSON はJSON保存用の構造体。
type entryJSON struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Thumbnail *string `json:"thumbnail,omitempty"`
	Text      string  `json:"text"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	Deleted   bool    `json:"deleted"`
}

func NewEntryStore(dataDir string) (*EntryStore, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	path := filepath.Join(dataDir, "entries.json")
	s := &EntryStore{
		path:    path,
		entries: make(map[uuid.UUID]domain.Entry),
	}

	if err := s.loadFromFile(); err != nil {
		return nil, fmt.Errorf("load entries: %w", err)
	}

	return s, nil
}

func (s *EntryStore) loadFromFile() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}

	var items []entryJSON
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}

	for _, item := range items {
		id, err := uuid.Parse(item.ID)
		if err != nil {
			continue
		}
		createdAt, _ := time.Parse(time.RFC3339Nano, item.CreatedAt)
		updatedAt, _ := time.Parse(time.RFC3339Nano, item.UpdatedAt)
		s.entries[id] = domain.Entry{
			ID:        id,
			Title:     item.Title,
			Content:   item.Content,
			Thumbnail: item.Thumbnail,
			Text:      item.Text,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			Deleted:   item.Deleted,
		}
	}
	return nil
}

func (s *EntryStore) saveToFile() error {
	items := make([]entryJSON, 0, len(s.entries))
	for _, entry := range s.entries {
		items = append(items, entryJSON{
			ID:        entry.ID.String(),
			Title:     entry.Title,
			Content:   entry.Content,
			Thumbnail: entry.Thumbnail,
			Text:      entry.Text,
			CreatedAt: entry.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt: entry.UpdatedAt.Format(time.RFC3339Nano),
			Deleted:   entry.Deleted,
		})
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0o644)
}

func (s *EntryStore) Save(_ context.Context, entry domain.Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.entries[entry.ID] = entry
	return s.saveToFile()
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
	slices.SortFunc(items, func(a, b domain.EntryListItem) int {
		return b.CreatedAt.Compare(a.CreatedAt)
	})
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
	return s.saveToFile()
}
