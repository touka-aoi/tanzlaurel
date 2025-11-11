package domain

import (
	"time"
)

// Meta はリクエスト共通のトレーシング情報を保持する。
type Meta struct {
	// RequestID はクライアントから渡された一意な識別子。
	RequestID string
	// TraceID は分散トレーシング用の識別子。
	TraceID string
	// OccurredAt はリクエストがクライアントで発生した時刻。
	OccurredAt time.Time
}
