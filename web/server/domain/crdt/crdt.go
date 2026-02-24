package crdt

import (
	"slices"
	"strings"

	"github.com/google/uuid"
)

// NodeID はRGA内のノードを一意に識別する。
type NodeID struct {
	ReplicaID    uuid.UUID `json:"replica_id"`
	Timestamp uint64    `json:"timestamp"`
}

// OpType はCRDTオペレーションの種別を表す。
type OpType int

const (
	OpInsert OpType = 1
	OpDelete OpType = 2
)

// Operation は冪等性をサポートするCRDTオペレーション。
type Operation struct {
	RequestID uuid.UUID `json:"request_id"`
	OpType    OpType    `json:"op_type"`
	NodeID    NodeID    `json:"node_id"`
	After     *NodeID   `json:"after"`
	Value     rune      `json:"value"`
}

// LamportClock はイベントの順序付けのための論理時計。
type LamportClock struct {
	replicaID  uuid.UUID
	counter uint64
}

// NewLamportClock は指定されたサイトの新しいLamportClockを作成する。
func NewLamportClock(replicaID uuid.UUID) *LamportClock {
	return &LamportClock{replicaID: replicaID, counter: 0}
}

// Tick はクロックをインクリメントし、新しいNodeIDを返す。
func (c *LamportClock) Tick() NodeID {
	c.counter++
	return NodeID{ReplicaID: c.replicaID, Timestamp: c.counter}
}

// Update は受信したタイムスタンプに基づいてクロックを更新する。
func (c *LamportClock) Update(ts uint64) {
	if ts > c.counter {
		c.counter = ts
	}
}

// ReplicaID はこのクロックのサイトIDを返す。
func (c *LamportClock) ReplicaID() uuid.UUID {
	return c.replicaID
}

// RGA はReplicated Growable Arrayを実装する。
type RGA struct {
	clock   *LamportClock
	nodes   []*node
	index   map[NodeID]int
	seen    map[uuid.UUID]struct{}
	pending []Operation // afterノードが未到着のオペレーションを保持するバッファ
}

type node struct {
	id      NodeID
	after   *NodeID
	value   rune
	deleted bool
}

// NewRGA は指定されたサイトの新しい空のRGAを作成する。
func NewRGA(replicaID uuid.UUID) *RGA {
	return &RGA{
		clock: NewLamportClock(replicaID),
		nodes: make([]*node, 0),
		index: make(map[NodeID]int),
		seen:  make(map[uuid.UUID]struct{}),
	}
}

// Apply はオペレーションをRGAに適用する。既に適用済みの場合はfalseを返す（冪等性）。
func (r *RGA) Apply(op Operation) bool {
	if _, exists := r.seen[op.RequestID]; exists {
		return false
	}
	r.seen[op.RequestID] = struct{}{}
	r.clock.Update(op.NodeID.Timestamp)

	switch op.OpType {
	case OpInsert:
		if op.After != nil {
			if _, ok := r.index[*op.After]; !ok {
				// afterノードが未到着 → バッファに追加
				r.pending = append(r.pending, op)
				return true
			}
		}
		r.applyInsert(op)
		r.flushPending()
	case OpDelete:
		if _, ok := r.index[op.NodeID]; !ok {
			// 対象ノードが未到着 → バッファに追加
			r.pending = append(r.pending, op)
			return true
		}
		r.applyDelete(op)
	}
	return true
}

// flushPending はバッファ内のオペレーションで適用可能なものを適用する。
func (r *RGA) flushPending() {
	for {
		applied := false
		remaining := r.pending[:0]
		for _, op := range r.pending {
			switch op.OpType {
			case OpInsert:
				if op.After != nil {
					if _, ok := r.index[*op.After]; !ok {
						remaining = append(remaining, op)
						continue
					}
				}
				r.applyInsert(op)
				applied = true
			case OpDelete:
				if _, ok := r.index[op.NodeID]; !ok {
					remaining = append(remaining, op)
					continue
				}
				r.applyDelete(op)
				applied = true
			}
		}
		r.pending = remaining
		if !applied {
			break
		}
	}
}

// nodeIDPriority は並行挿入の優先順位を比較する。
// trueを返す場合、aはbより左（先）に配置される。
// ルール: Timestamp大 → 左、同Timestampなら ReplicaID辞書順小 → 左
func nodeIDPriority(a, b NodeID) bool {
	if a.Timestamp != b.Timestamp {
		return a.Timestamp > b.Timestamp
	}
	return a.ReplicaID.String() < b.ReplicaID.String()
}

// applyInsert はInsertオペレーションを内部的に適用する。
func (r *RGA) applyInsert(op Operation) {
	n := &node{
		id:    op.NodeID,
		after: op.After,
		value: op.Value,
	}

	// 挿入位置を決定（afterノードの直後から探索開始）
	insertIdx := 0
	if op.After != nil {
		insertIdx = r.index[*op.After] + 1
	}
	for insertIdx < len(r.nodes) {
		existing := r.nodes[insertIdx]

		// 同じafterを共有する兄弟ノードかチェック
		if !sameAfter(existing.after, op.After) {
			break
		}

		// 兄弟ノード間での優先順位比較
		if nodeIDPriority(op.NodeID, existing.id) {
			break
		}

		// 既存ノードの方が優先度が高い → その子孫もスキップ
		insertIdx = r.skipSubtree(insertIdx)
	}

	// ノードを挿入
	r.nodes = slices.Insert(r.nodes, insertIdx, n)

	// インデックスを再構築
	for i := insertIdx; i < len(r.nodes); i++ {
		r.index[r.nodes[i].id] = i
	}
}

// skipSubtree は指定インデックスのノードとその子孫をスキップし、次の兄弟の位置を返す。
func (r *RGA) skipSubtree(idx int) int {
	parentID := r.nodes[idx].id
	idx++
	for idx < len(r.nodes) {
		if !r.isDescendantOf(idx, parentID) {
			break
		}
		idx++
	}
	return idx
}

// isDescendantOf は指定インデックスのノードがparentIDの子孫かどうかを判定する。
func (r *RGA) isDescendantOf(idx int, parentID NodeID) bool {
	n := r.nodes[idx]
	if n.after == nil {
		return false
	}
	if *n.after == parentID {
		return true
	}
	ancestorIdx, ok := r.index[*n.after]
	if !ok {
		return false
	}
	return r.isDescendantOf(ancestorIdx, parentID)
}

// sameAfter は2つのafterポインタが同じかどうかを比較する。
func sameAfter(a, b *NodeID) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// applyDelete はDeleteオペレーションを内部的に適用する（トゥームストーン方式）。
func (r *RGA) applyDelete(op Operation) {
	idx, ok := r.index[op.NodeID]
	if !ok {
		return
	}
	r.nodes[idx].deleted = true
}

// Text はRGAの現在のテキスト内容を返す。
func (r *RGA) Text() string {
	var sb strings.Builder
	for _, n := range r.nodes {
		if !n.deleted {
			sb.WriteRune(n.value)
		}
	}
	return sb.String()
}

// NodeCount はトゥームストーン含む全ノード数を返す。
func (r *RGA) NodeCount() int {
	return len(r.nodes)
}

// Insert は挿入オペレーションを作成して適用する。オペレーションを返す。
func (r *RGA) Insert(after *NodeID, value rune) Operation {
	op := Operation{
		RequestID: uuid.New(),
		OpType:    OpInsert,
		NodeID:    r.clock.Tick(),
		After:     after,
		Value:     value,
	}
	r.Apply(op)
	return op
}

// Delete は削除オペレーションを作成して適用する。オペレーションを返す。
func (r *RGA) Delete(nodeID NodeID) Operation {
	op := Operation{
		RequestID: uuid.New(),
		OpType:    OpDelete,
		NodeID:    nodeID,
	}
	r.Apply(op)
	return op
}
