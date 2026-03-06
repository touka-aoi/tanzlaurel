package jsonfile_test

import (
	"encoding/json"
	"testing"

	"flourish/server/adapter/jsonfile"
	"flourish/server/domain"

	"github.com/google/uuid"
)

func TestEventStore_AppendAndList(t *testing.T) {
	dir := t.TempDir()
	store, err := jsonfile.NewEventStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	entryID := uuid.New()
	reqID := uuid.New()
	payload, _ := json.Marshal(map[string]any{"op_type": 1, "value": "a"})

	seq, err := store.Append(t.Context(), domain.Event{
		EntryID:   entryID,
		RequestID: reqID,
		EventType: domain.EventCRDTOp,
		SiteID:    uuid.New(),
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 {
		t.Errorf("seq: got %d, want 1", seq)
	}

	// 重複
	seq2, err := store.Append(t.Context(), domain.Event{
		EntryID:   entryID,
		RequestID: reqID,
		EventType: domain.EventCRDTOp,
		Payload:   payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq2 != 0 {
		t.Errorf("duplicate seq: got %d, want 0", seq2)
	}

	// ListAfter
	events, err := store.ListAfter(t.Context(), entryID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Errorf("events count: got %d, want 1", len(events))
	}

	// MaxServerSeq
	maxSeq, err := store.MaxServerSeq(t.Context(), entryID)
	if err != nil {
		t.Fatal(err)
	}
	if maxSeq != 1 {
		t.Errorf("maxSeq: got %d, want 1", maxSeq)
	}
}

func TestEventStore_PersistAndReload(t *testing.T) {
	dir := t.TempDir()

	entryID := uuid.New()
	reqID1 := uuid.New()
	reqID2 := uuid.New()

	// 1回目: 2件書き込み
	{
		store, err := jsonfile.NewEventStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		payload, _ := json.Marshal(map[string]any{"value": "a"})
		store.Append(t.Context(), domain.Event{
			EntryID: entryID, RequestID: reqID1, EventType: domain.EventCRDTOp, Payload: payload,
		})
		store.Append(t.Context(), domain.Event{
			EntryID: entryID, RequestID: reqID2, EventType: domain.EventCRDTOp, Payload: payload,
		})
	}

	// 2回目: 再読み込み
	{
		store, err := jsonfile.NewEventStore(dir)
		if err != nil {
			t.Fatal(err)
		}

		maxSeq, _ := store.MaxServerSeq(t.Context(), entryID)
		if maxSeq != 2 {
			t.Errorf("reloaded maxSeq: got %d, want 2", maxSeq)
		}

		events, _ := store.ListAfter(t.Context(), entryID, 0)
		if len(events) != 2 {
			t.Errorf("reloaded events: got %d, want 2", len(events))
		}

		// 重複検知も復元されている
		seq, _ := store.Append(t.Context(), domain.Event{
			EntryID: entryID, RequestID: reqID1, EventType: domain.EventCRDTOp,
		})
		if seq != 0 {
			t.Errorf("dedup after reload: got %d, want 0", seq)
		}

		// 新規は seq=3
		seq, _ = store.Append(t.Context(), domain.Event{
			EntryID: entryID, RequestID: uuid.New(), EventType: domain.EventCRDTOp,
			Payload: []byte(`{}`),
		})
		if seq != 3 {
			t.Errorf("new seq after reload: got %d, want 3", seq)
		}
	}
}
