package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventType はイベントの種別を表す。
type EventType string

const (
	EventCRDTOp      EventType = "crdt_op"
	EventEntryCreate EventType = "entry_create"
	EventEntryDelete EventType = "entry_delete"
)

// Event はイベントストアに保存されるイベントを表す。
type Event struct {
	EntryID   uuid.UUID
	ServerSeq int64
	RequestID uuid.UUID
	EventType EventType
	SiteID    uuid.UUID
	Payload   []byte
	CreatedAt time.Time
}
