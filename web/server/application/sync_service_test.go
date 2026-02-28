package application_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"flourish/server/adapter/memory"
	"flourish/server/application"
	"flourish/server/domain"
	"flourish/server/domain/crdt"

	"github.com/google/uuid"
)

type mockSubscriber struct {
	mu       sync.Mutex
	messages []application.SyncMessage
}

func (s *mockSubscriber) Send(msg application.SyncMessage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, msg)
}

func (s *mockSubscriber) Messages() []application.SyncMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]application.SyncMessage{}, s.messages...)
}

func TestSyncService_HandleOp_PersistsAndBroadcasts(t *testing.T) {
	eventStore := memory.NewEventStore()
	svc := application.NewSyncService(eventStore)
	ctx := context.Background()

	entryID := uuid.New()
	siteID := uuid.New()

	sub := &mockSubscriber{}
	svc.Subscribe(entryID, sub)
	defer svc.Unsubscribe(entryID, sub)

	op := crdt.Operation{
		RequestID: uuid.New(),
		OpType:    crdt.OpInsert,
		NodeID:    crdt.NodeID{ReplicaID: siteID, Timestamp: 1},
		Value:     'a',
	}
	payload, _ := json.Marshal(op)

	ack, err := svc.HandleOp(ctx, entryID, siteID, op.RequestID, payload)
	if err != nil {
		t.Fatal(err)
	}
	if ack.ServerSeq != 1 {
		t.Errorf("ServerSeqは1であるべき: got %d", ack.ServerSeq)
	}

	// イベントが永続化されているか確認
	events, _ := eventStore.ListAfter(ctx, entryID, 0)
	if len(events) != 1 {
		t.Fatalf("イベントが1件であるべき: got %d", len(events))
	}
	if events[0].EventType != domain.EventCRDTOp {
		t.Errorf("EventTypeがcrdt_opであるべき: got %q", events[0].EventType)
	}

	// subscriberにbroadcastされているか確認
	msgs := sub.Messages()
	if len(msgs) != 1 {
		t.Fatalf("broadcastが1件であるべき: got %d", len(msgs))
	}
	if msgs[0].LatestServerSeq != 1 {
		t.Errorf("LatestServerSeqは1であるべき: got %d", msgs[0].LatestServerSeq)
	}
}

func TestSyncService_HandleOp_Dedup(t *testing.T) {
	eventStore := memory.NewEventStore()
	svc := application.NewSyncService(eventStore)
	ctx := context.Background()

	entryID := uuid.New()
	siteID := uuid.New()
	reqID := uuid.New()

	op := crdt.Operation{
		RequestID: reqID,
		OpType:    crdt.OpInsert,
		NodeID:    crdt.NodeID{ReplicaID: siteID, Timestamp: 1},
		Value:     'a',
	}
	payload, _ := json.Marshal(op)

	ack1, _ := svc.HandleOp(ctx, entryID, siteID, reqID, payload)
	if ack1.ServerSeq != 1 {
		t.Errorf("最初のopのServerSeqは1: got %d", ack1.ServerSeq)
	}

	// 同じrequest_idで再送
	ack2, _ := svc.HandleOp(ctx, entryID, siteID, reqID, payload)
	if ack2.ServerSeq != 0 {
		t.Errorf("重複opのServerSeqは0: got %d", ack2.ServerSeq)
	}
}

func TestSyncService_GetDiff(t *testing.T) {
	eventStore := memory.NewEventStore()
	svc := application.NewSyncService(eventStore)
	ctx := context.Background()

	entryID := uuid.New()
	siteID := uuid.New()

	// 3件のopを登録
	for i := range 3 {
		op := crdt.Operation{
			RequestID: uuid.New(),
			OpType:    crdt.OpInsert,
			NodeID:    crdt.NodeID{ReplicaID: siteID, Timestamp: uint64(i + 1)},
			Value:     rune('a' + i),
		}
		payload, _ := json.Marshal(op)
		svc.HandleOp(ctx, entryID, siteID, op.RequestID, payload)
	}

	// seq=1以降の差分を取得
	msg, err := svc.GetDiff(ctx, entryID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.Ops) != 2 {
		t.Errorf("差分opは2件であるべき: got %d", len(msg.Ops))
	}
	if msg.LatestServerSeq != 3 {
		t.Errorf("LatestServerSeqは3であるべき: got %d", msg.LatestServerSeq)
	}
}
