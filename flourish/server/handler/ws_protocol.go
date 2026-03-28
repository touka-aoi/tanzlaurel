package handler

import "github.com/google/uuid"

// WebSocketメッセージ型
const (
	MsgTypeOp          = "op"
	MsgTypeAck         = "ack"
	MsgTypeSyncRequest = "sync_request"
	MsgTypeSync        = "sync"
	MsgTypeError       = "error"
)

// NodeIDMsg はNodeIDのJSON表現。
type NodeIDMsg struct {
	SiteID    string `json:"site_id"`
	Timestamp uint64 `json:"timestamp"`
}

// IncomingMessage はクライアントから受信するメッセージの共通構造。
type IncomingMessage struct {
	Type          string     `json:"type"`
	RequestID     string     `json:"request_id"`
	EntryID       string     `json:"entry_id"`
	OpType        int        `json:"op_type,omitempty"`
	NodeID        *NodeIDMsg `json:"node_id,omitempty"`
	After         *NodeIDMsg `json:"after,omitempty"`
	Value         string     `json:"value,omitempty"`
	LastServerSeq int64      `json:"last_server_seq,omitempty"`
	Authenticated *bool      `json:"authenticated,omitempty"`
}

// AckMsg はACKレスポンス。
type AckMsg struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	EntryID   string `json:"entry_id"`
	ServerSeq int64  `json:"server_seq"`
}

// SyncOpMsg はsync内の個別op。
type SyncOpMsg struct {
	RequestID     string     `json:"request_id"`
	ServerSeq     int64      `json:"server_seq"`
	OpType        int        `json:"op_type"`
	NodeID        *NodeIDMsg `json:"node_id"`
	After         *NodeIDMsg `json:"after,omitempty"`
	Value         string     `json:"value,omitempty"`
	Authenticated *bool      `json:"authenticated,omitempty"`
}

// SyncMsg はsyncメッセージ。
type SyncMsg struct {
	Type            string      `json:"type"`
	EntryID         string      `json:"entry_id"`
	Ops             []SyncOpMsg `json:"ops"`
	LatestServerSeq int64       `json:"latest_server_seq"`
}

// ErrorMsg はエラーメッセージ。
type ErrorMsg struct {
	Type      string  `json:"type"`
	RequestID *string `json:"request_id"`
	ErrorType string  `json:"error_type"`
	Title     string  `json:"title"`
	Instance  string  `json:"instance"`
}

func newErrorMsg(requestID *string, errorType, title string) ErrorMsg {
	return ErrorMsg{
		Type:      MsgTypeError,
		RequestID: requestID,
		ErrorType: errorType,
		Title:     title,
		Instance:  "urn:uuid:" + uuid.New().String(),
	}
}
