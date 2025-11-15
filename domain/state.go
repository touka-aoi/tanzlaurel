package domain

import "time"

// PlayerSnapshot はアプリケーションが扱うプレイヤーの状態。
type PlayerSnapshot struct {
	PlayerID string
	RoomID   string
	Position Vec2
	Health   int

	ActiveBuffs []ActiveBuff
	Inventory   []InventoryEntry

	LastUpdated time.Time
}

// ActiveBuff はプレイヤーに付与されているバフと有効期限。
type ActiveBuff struct {
	Buff      Buff
	ExpiresAt time.Time
}

// InventoryEntry はインベントリ内のアイテムと個数。
type InventoryEntry struct {
	ItemID   string
	Quantity int
}

// RoomSnapshot はルーム全体の状態。
type RoomSnapshot struct {
	RoomID      string
	MemberIDs   []string
	LastUpdated time.Time
}

// RoomStats はルーム単位での集計値を表す。
type RoomStats struct {
	RoomID string

	TotalHealth         int
	AverageEnergy       float64
	ActiveBuffHistogram map[string]int
	InteractionCount    int
	LastUpdated         time.Time
}

// MoveResult は移動処理結果。
type MoveResult struct {
	Player PlayerSnapshot
}

// BuffResult はバフ適用後の結果。
type BuffResult struct {
	AffectedPlayers []PlayerSnapshot
	Room            RoomSnapshot
	Stats           RoomStats
}

// AttackResult は攻撃処理の結果。
type AttackResult struct {
	Attacker PlayerSnapshot
	Target   PlayerSnapshot
	Room     RoomSnapshot
	Stats    RoomStats
}

// TradeLedger は取引処理の整合性記録。
type TradeLedger struct {
	Confirmed bool
	Initiator PlayerSnapshot
	Partner   PlayerSnapshot
}

// TradeResult はトレード処理の結果。
type TradeResult struct {
	Initiator PlayerSnapshot
	Partner   PlayerSnapshot
	Ledger    TradeLedger
	Stats     RoomStats
}
