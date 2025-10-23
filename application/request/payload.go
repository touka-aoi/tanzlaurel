package request

import (
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
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

// Move は HTTP レイヤから渡される移動リクエスト。
type Move struct {
	Meta    Meta
	Command domain.MoveCommand
}

// Buff は複数プレイヤーにバフを適用するリクエスト。
type Buff struct {
	Meta    Meta
	Command domain.BuffCommand
}

// Attack は攻撃処理を要求するリクエスト。
type Attack struct {
	Meta    Meta
	Command domain.AttackCommand
}

// Trade はトレード処理を要求するリクエスト。
type Trade struct {
	Meta    Meta
	Command domain.TradeCommand
}
