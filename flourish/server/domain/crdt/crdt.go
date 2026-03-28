package crdt

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

// payloadMsg はIncomingMessageのPayloadから必要フィールドを抽出する構造体。
type payloadMsg struct {
	RequestID     string      `json:"request_id"`
	OpType        int         `json:"op_type"`
	NodeID        *payloadNID `json:"node_id"`
	After         *payloadNID `json:"after"`
	Value         string      `json:"value"`
	Authenticated *bool       `json:"authenticated,omitempty"`
}

type payloadNID struct {
	SiteID    string `json:"site_id"`
	Timestamp uint64 `json:"timestamp"`
}

// OperationFromPayload はIncomingMessageのJSONバイト列からOperationを生成する。
func OperationFromPayload(payload []byte) (Operation, error) {
	var msg payloadMsg
	if err := json.Unmarshal(payload, &msg); err != nil {
		return Operation{}, fmt.Errorf("unmarshal payload: %w", err)
	}

	requestID, err := uuid.Parse(msg.RequestID)
	if err != nil {
		return Operation{}, fmt.Errorf("parse request_id: %w", err)
	}

	if msg.NodeID == nil {
		return Operation{}, fmt.Errorf("node_id is required")
	}

	replicaID, err := uuid.Parse(msg.NodeID.SiteID)
	if err != nil {
		return Operation{}, fmt.Errorf("parse node_id.site_id: %w", err)
	}

	op := Operation{
		RequestID: requestID,
		OpType:    OpType(msg.OpType),
		NodeID: NodeID{
			ReplicaID: replicaID,
			Timestamp: msg.NodeID.Timestamp,
		},
	}

	if msg.After != nil {
		afterReplicaID, err := uuid.Parse(msg.After.SiteID)
		if err != nil {
			return Operation{}, fmt.Errorf("parse after.site_id: %w", err)
		}
		after := NodeID{
			ReplicaID: afterReplicaID,
			Timestamp: msg.After.Timestamp,
		}
		op.After = &after
	}

	if msg.Value != "" {
		r, _ := utf8.DecodeRuneInString(msg.Value)
		op.Value = r
	}

	// authenticated: 明示的にfalseが指定されない限りtrue（既存データ互換）
	if msg.Authenticated != nil {
		op.Authenticated = *msg.Authenticated
	} else {
		op.Authenticated = true
	}

	return op, nil
}

// RGASnapshot はRGAの永続化用構造体。
type RGASnapshot struct {
	ReplicaID string         `json:"replica_id"`
	Counter   uint64         `json:"counter"`
	Nodes     []NodeSnapshot `json:"nodes"`
	Seen      []string       `json:"seen"`
	Pending   []Operation    `json:"pending,omitempty"`
}

// NodeSnapshot はノードの永続化用構造体。
type NodeSnapshot struct {
	ID            NodeID  `json:"id"`
	After         *NodeID `json:"after"`
	Value         string  `json:"value"`
	Deleted       bool    `json:"deleted"`
	Authenticated *bool   `json:"authenticated,omitempty"`
}

// Export はRGAをシリアライズ可能なスナップショットに変換する。
func (r *RGA) Export() RGASnapshot {
	nodes := make([]NodeSnapshot, len(r.nodes))
	for i, n := range r.nodes {
		auth := n.authenticated
		nodes[i] = NodeSnapshot{
			ID:            n.id,
			After:         n.after,
			Value:         string(n.value),
			Deleted:       n.deleted,
			Authenticated: &auth,
		}
	}

	seen := make([]string, 0, len(r.seen))
	for id := range r.seen {
		seen = append(seen, id.String())
	}

	// pendingをコピー
	pending := make([]Operation, len(r.pending))
	copy(pending, r.pending)

	return RGASnapshot{
		ReplicaID: r.clock.ReplicaID().String(),
		Counter:   r.clock.counter,
		Nodes:     nodes,
		Seen:      seen,
		Pending:   pending,
	}
}

// ImportRGA はスナップショットからRGAを復元する。
func ImportRGA(snap RGASnapshot) (*RGA, error) {
	replicaID, err := uuid.Parse(snap.ReplicaID)
	if err != nil {
		return nil, fmt.Errorf("parse replica_id: %w", err)
	}

	rga := &RGA{
		clock: &LamportClock{
			replicaID: replicaID,
			counter:   snap.Counter,
		},
		nodes: make([]*node, len(snap.Nodes)),
		index: make(map[NodeID]int, len(snap.Nodes)),
		seen:  make(map[uuid.UUID]struct{}, len(snap.Seen)),
	}

	for i, ns := range snap.Nodes {
		r, _ := utf8.DecodeRuneInString(ns.Value)
		// 既存データ（Authenticated未設定=nil）は認証済みとして扱う
		auth := true
		if ns.Authenticated != nil {
			auth = *ns.Authenticated
		}
		rga.nodes[i] = &node{
			id:            ns.ID,
			after:         ns.After,
			value:         r,
			deleted:       ns.Deleted,
			authenticated: auth,
		}
		rga.index[ns.ID] = i
	}

	for _, s := range snap.Seen {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, fmt.Errorf("parse seen id: %w", err)
		}
		rga.seen[id] = struct{}{}
	}

	// pending復元
	rga.pending = make([]Operation, len(snap.Pending))
	copy(rga.pending, snap.Pending)

	return rga, nil
}

// NodeID はRGA内のノードを一意に識別する。
type NodeID struct {
	ReplicaID uuid.UUID `json:"replica_id"`
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
	RequestID     uuid.UUID `json:"request_id"`
	OpType        OpType    `json:"op_type"`
	NodeID        NodeID    `json:"node_id"`
	After         *NodeID   `json:"after"`
	Value         rune      `json:"value"`
	Authenticated bool      `json:"authenticated"`
}

// LamportClock はイベントの順序付けのための論理時計。
type LamportClock struct {
	replicaID uuid.UUID
	counter   uint64
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
	id            NodeID
	after         *NodeID
	value         rune
	deleted       bool
	authenticated bool
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
		id:            op.NodeID,
		after:         op.After,
		value:         op.Value,
		authenticated: op.Authenticated,
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

// IsNodeAuthenticated は指定ノードが認証済みかどうかを返す。
// ノードが存在しない場合はtrueを返す（安全側に倒す）。
func (r *RGA) IsNodeAuthenticated(id NodeID) bool {
	idx, ok := r.index[id]
	if !ok {
		return true
	}
	return r.nodes[idx].authenticated
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

// TextFiltered はexcludeに含まれるノードIDの文字をスキップしたテキストを返す。
func (r *RGA) TextFiltered(exclude map[NodeID]struct{}) string {
	var sb strings.Builder
	for _, n := range r.nodes {
		if _, ok := exclude[n.id]; ok {
			continue
		}
		if n.deleted {
			continue
		}
		sb.WriteRune(n.value)
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
