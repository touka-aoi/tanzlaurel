package crdt_test

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"flourish/server/domain/crdt"
)

func TestOperationFromPayload(t *testing.T) {
	siteID := uuid.New()
	reqID := uuid.New()
	afterSiteID := uuid.New()

	payload, _ := json.Marshal(map[string]any{
		"type":       "op",
		"request_id": reqID.String(),
		"op_type":    1,
		"node_id":    map[string]any{"site_id": siteID.String(), "timestamp": 5},
		"after":      map[string]any{"site_id": afterSiteID.String(), "timestamp": 3},
		"value":      "a",
	})

	op, err := crdt.OperationFromPayload(payload)
	if err != nil {
		t.Fatal(err)
	}

	if op.RequestID != reqID {
		t.Errorf("RequestID: got %v, want %v", op.RequestID, reqID)
	}
	if op.OpType != crdt.OpInsert {
		t.Errorf("OpType: got %v, want %v", op.OpType, crdt.OpInsert)
	}
	if op.NodeID.ReplicaID != siteID {
		t.Errorf("NodeID.ReplicaID: got %v, want %v", op.NodeID.ReplicaID, siteID)
	}
	if op.NodeID.Timestamp != 5 {
		t.Errorf("NodeID.Timestamp: got %d, want 5", op.NodeID.Timestamp)
	}
	if op.After == nil {
		t.Fatal("After should not be nil")
	}
	if op.After.ReplicaID != afterSiteID {
		t.Errorf("After.ReplicaID: got %v, want %v", op.After.ReplicaID, afterSiteID)
	}
	if op.After.Timestamp != 3 {
		t.Errorf("After.Timestamp: got %d, want 3", op.After.Timestamp)
	}
	if op.Value != 'a' {
		t.Errorf("Value: got %c, want 'a'", op.Value)
	}
}

func TestOperationFromPayload_Delete(t *testing.T) {
	siteID := uuid.New()
	reqID := uuid.New()

	payload, _ := json.Marshal(map[string]any{
		"request_id": reqID.String(),
		"op_type":    2,
		"node_id":    map[string]any{"site_id": siteID.String(), "timestamp": 3},
	})

	op, err := crdt.OperationFromPayload(payload)
	if err != nil {
		t.Fatal(err)
	}

	if op.OpType != crdt.OpDelete {
		t.Errorf("OpType: got %v, want %v", op.OpType, crdt.OpDelete)
	}
	if op.After != nil {
		t.Error("Delete op should not have After")
	}
}

func TestRGA_ExportImport(t *testing.T) {
	rga := crdt.NewRGA(uuid.New())

	// テキスト "hello" を挿入
	var lastNodeID *crdt.NodeID
	for _, ch := range "hello" {
		op := rga.Insert(lastNodeID, ch)
		nid := op.NodeID
		lastNodeID = &nid
	}

	if rga.Text() != "hello" {
		t.Fatalf("Text before export: got %q, want %q", rga.Text(), "hello")
	}

	// Export
	snap := rga.Export()

	// Import
	restored, err := crdt.ImportRGA(snap)
	if err != nil {
		t.Fatal(err)
	}

	if restored.Text() != "hello" {
		t.Errorf("Text after import: got %q, want %q", restored.Text(), "hello")
	}
	if restored.NodeCount() != rga.NodeCount() {
		t.Errorf("NodeCount: got %d, want %d", restored.NodeCount(), rga.NodeCount())
	}

	// Importしたものに追加挿入できるか
	restored.Insert(lastNodeID, '!')
	if restored.Text() != "hello!" {
		t.Errorf("Text after additional insert: got %q, want %q", restored.Text(), "hello!")
	}
}

func TestRGA_ExportImport_WithDeletes(t *testing.T) {
	rga := crdt.NewRGA(uuid.New())

	op1 := rga.Insert(nil, 'a')
	nid1 := op1.NodeID
	op2 := rga.Insert(&nid1, 'b')
	nid2 := op2.NodeID
	rga.Insert(&nid2, 'c')
	rga.Delete(nid2) // 'b' を削除

	if rga.Text() != "ac" {
		t.Fatalf("Text before export: got %q, want %q", rga.Text(), "ac")
	}

	snap := rga.Export()
	restored, err := crdt.ImportRGA(snap)
	if err != nil {
		t.Fatal(err)
	}

	if restored.Text() != "ac" {
		t.Errorf("Text after import: got %q, want %q", restored.Text(), "ac")
	}
}

func TestRGA_ExportImport_Idempotency(t *testing.T) {
	rga := crdt.NewRGA(uuid.New())
	op := rga.Insert(nil, 'x')

	snap := rga.Export()
	restored, err := crdt.ImportRGA(snap)
	if err != nil {
		t.Fatal(err)
	}

	// 同じopをapplyしても重複しない（冪等性が保たれる）
	if restored.Apply(op) {
		t.Error("Apply should return false for duplicate op")
	}
	if restored.Text() != "x" {
		t.Errorf("Text: got %q, want %q", restored.Text(), "x")
	}
}
