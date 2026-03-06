package memory_test

import (
	"context"
	"testing"

	"flourish/server/adapter/memory"
	"flourish/server/domain"

	"github.com/google/uuid"
)

func TestEventStore_Append_AssignsSeq(t *testing.T) {
	store := memory.NewEventStore()
	ctx := context.Background()
	entryID := uuid.New()

	seq, err := store.Append(ctx, domain.Event{
		EntryID:   entryID,
		RequestID: uuid.New(),
		EventType: domain.EventCRDTOp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 {
		t.Errorf("最初のseqは1であるべき: got %d", seq)
	}

	seq2, err := store.Append(ctx, domain.Event{
		EntryID:   entryID,
		RequestID: uuid.New(),
		EventType: domain.EventCRDTOp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq2 != 2 {
		t.Errorf("2番目のseqは2であるべき: got %d", seq2)
	}
}

func TestEventStore_Append_Dedup(t *testing.T) {
	store := memory.NewEventStore()
	ctx := context.Background()
	reqID := uuid.New()

	seq, err := store.Append(ctx, domain.Event{
		EntryID:   uuid.New(),
		RequestID: reqID,
		EventType: domain.EventCRDTOp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq != 1 {
		t.Errorf("最初のseqは1であるべき: got %d", seq)
	}

	seq2, err := store.Append(ctx, domain.Event{
		EntryID:   uuid.New(),
		RequestID: reqID,
		EventType: domain.EventCRDTOp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq2 != 0 {
		t.Errorf("重複request_idの場合は0を返すべき: got %d", seq2)
	}
}

func TestEventStore_ListAfter(t *testing.T) {
	store := memory.NewEventStore()
	ctx := context.Background()
	entryID := uuid.New()

	for i := 0; i < 5; i++ {
		store.Append(ctx, domain.Event{
			EntryID:   entryID,
			RequestID: uuid.New(),
			EventType: domain.EventCRDTOp,
		})
	}

	events, err := store.ListAfter(ctx, entryID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Errorf("seq>3のイベントは2件: got %d", len(events))
	}
	if events[0].ServerSeq != 4 {
		t.Errorf("最初のイベントのseqは4: got %d", events[0].ServerSeq)
	}
}

func TestEventStore_MaxServerSeq(t *testing.T) {
	store := memory.NewEventStore()
	ctx := context.Background()
	entryID := uuid.New()

	maxSeq, err := store.MaxServerSeq(ctx, entryID)
	if err != nil {
		t.Fatal(err)
	}
	if maxSeq != 0 {
		t.Errorf("空のストアのMaxServerSeqは0: got %d", maxSeq)
	}

	for i := 0; i < 3; i++ {
		store.Append(ctx, domain.Event{
			EntryID:   entryID,
			RequestID: uuid.New(),
			EventType: domain.EventCRDTOp,
		})
	}

	maxSeq, err = store.MaxServerSeq(ctx, entryID)
	if err != nil {
		t.Fatal(err)
	}
	if maxSeq != 3 {
		t.Errorf("3件追加後のMaxServerSeqは3: got %d", maxSeq)
	}
}
