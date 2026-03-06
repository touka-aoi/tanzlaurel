package application_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"flourish/server/application"
	"flourish/server/domain"
	"flourish/server/domain/crdt"

	"github.com/google/uuid"
)

// mockEntryStore はテスト用のEntryStore。
type mockEntryStore struct {
	entries map[uuid.UUID]domain.Entry
}

func newMockEntryStore() *mockEntryStore {
	return &mockEntryStore{entries: make(map[uuid.UUID]domain.Entry)}
}

func (s *mockEntryStore) Save(_ context.Context, entry domain.Entry) error {
	s.entries[entry.ID] = entry
	return nil
}

func (s *mockEntryStore) FindByID(_ context.Context, id uuid.UUID) (domain.Entry, error) {
	e, ok := s.entries[id]
	if !ok {
		return domain.Entry{}, domain.ErrEntryNotFound
	}
	return e, nil
}

func (s *mockEntryStore) List(_ context.Context) ([]domain.EntryListItem, error) {
	return nil, nil
}

func (s *mockEntryStore) Delete(_ context.Context, id uuid.UUID) error {
	return nil
}

// mockRGAStateStore はテスト用のRGAStateStore。
type mockRGAStateStore struct {
	states map[uuid.UUID]crdt.RGASnapshot
}

func newMockRGAStateStore() *mockRGAStateStore {
	return &mockRGAStateStore{states: make(map[uuid.UUID]crdt.RGASnapshot)}
}

func (s *mockRGAStateStore) SaveRGA(_ context.Context, entryID uuid.UUID, snap crdt.RGASnapshot) error {
	s.states[entryID] = snap
	return nil
}

func (s *mockRGAStateStore) LoadRGA(_ context.Context, entryID uuid.UUID) (crdt.RGASnapshot, error) {
	snap, ok := s.states[entryID]
	if !ok {
		return crdt.RGASnapshot{}, domain.ErrEntryNotFound
	}
	return snap, nil
}

func (s *mockRGAStateStore) ListRGAEntryIDs(_ context.Context) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(s.states))
	for id := range s.states {
		ids = append(ids, id)
	}
	return ids, nil
}

func makeInsertPayload(t *testing.T, siteID uuid.UUID, timestamp uint64, value string, after *struct{ SiteID uuid.UUID; Timestamp uint64 }) []byte {
	t.Helper()
	msg := map[string]any{
		"type":       "op",
		"request_id": uuid.New().String(),
		"op_type":    1,
		"node_id":    map[string]any{"site_id": siteID.String(), "timestamp": timestamp},
		"value":      value,
	}
	if after != nil {
		msg["after"] = map[string]any{"site_id": after.SiteID.String(), "timestamp": after.Timestamp}
	}
	data, _ := json.Marshal(msg)
	return data
}

func makeDeletePayload(t *testing.T, siteID uuid.UUID, timestamp uint64) []byte {
	t.Helper()
	msg := map[string]any{
		"type":       "op",
		"request_id": uuid.New().String(),
		"op_type":    2,
		"node_id":    map[string]any{"site_id": siteID.String(), "timestamp": timestamp},
	}
	data, _ := json.Marshal(msg)
	return data
}

func TestEntryProjector_Apply(t *testing.T) {
	entryStore := newMockEntryStore()
	rgaStore := newMockRGAStateStore()
	projector := application.NewEntryProjector(entryStore, rgaStore, t.TempDir(), nil)

	entryID := uuid.New()
	siteID := uuid.New()

	// エントリを事前に作成
	entryStore.entries[entryID] = domain.Entry{
		ID:        entryID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// 'H' を挿入
	payload1 := makeInsertPayload(t, siteID, 1, "H", nil)
	projector.Apply(context.Background(), entryID, payload1)

	entry := entryStore.entries[entryID]
	if entry.Title != "H" {
		t.Errorf("Title: got %q, want %q", entry.Title, "H")
	}
	if entry.Text != "H" {
		t.Errorf("Text: got %q, want %q", entry.Text, "H")
	}

	// 'i' をHの後に挿入
	afterH := &struct{ SiteID uuid.UUID; Timestamp uint64 }{siteID, 1}
	payload2 := makeInsertPayload(t, siteID, 2, "i", afterH)
	projector.Apply(context.Background(), entryID, payload2)

	entry = entryStore.entries[entryID]
	if entry.Text != "Hi" {
		t.Errorf("Text: got %q, want %q", entry.Text, "Hi")
	}

	// RGAスナップショットが保存されていること
	if _, ok := rgaStore.states[entryID]; !ok {
		t.Error("RGA snapshot should be saved")
	}
}

func TestEntryProjector_TitleFromFirstLine(t *testing.T) {
	entryStore := newMockEntryStore()
	rgaStore := newMockRGAStateStore()
	projector := application.NewEntryProjector(entryStore, rgaStore, t.TempDir(), nil)

	entryID := uuid.New()
	siteID := uuid.New()

	entryStore.entries[entryID] = domain.Entry{
		ID:        entryID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// "AB\nCD" を挿入
	chars := []struct{ ch string; ts uint64 }{
		{"A", 1}, {"B", 2}, {"\n", 3}, {"C", 4}, {"D", 5},
	}
	for i, c := range chars {
		var after *struct{ SiteID uuid.UUID; Timestamp uint64 }
		if i > 0 {
			after = &struct{ SiteID uuid.UUID; Timestamp uint64 }{siteID, chars[i-1].ts}
		}
		payload := makeInsertPayload(t, siteID, c.ts, c.ch, after)
		projector.Apply(context.Background(), entryID, payload)
	}

	entry := entryStore.entries[entryID]
	if entry.Title != "AB" {
		t.Errorf("Title: got %q, want %q", entry.Title, "AB")
	}
	if entry.Text != "AB\nCD" {
		t.Errorf("Text: got %q, want %q", entry.Text, "AB\nCD")
	}
}
