package domain

import "time"

// Vec2 は 2 次元座標を表す値オブジェクト。
type Vec2 struct {
	X float64
	Y float64
}

// BuffEffect はバフ効果の識別子と量、持続時間を表す。
type Buff struct {
	BuffID   string
	Value    float64
	Duration time.Duration
}

// ItemChange はインベントリの増減を表すドメイン値オブジェクト。
type ItemChange struct {
	ItemID        string
	QuantityDelta int
}

// MoveCommand は移動処理に必要なドメインコマンド。
type MoveCommand struct {
	UserID       string
	NextPosition Vec2
	Facing       float64 // 向き
}

// BuffCommand はバフ適用処理のドメインコマンド。
type BuffCommand struct {
	UserID    string
	TargetIDs []string
	Buff      Buff
}

// AttackCommand は攻撃処理のドメインコマンド。
type AttackCommand struct {
	UserID   string
	TargetID string
	Damage   int
}

// TradeCommand はトレード処理のドメインコマンド。
type TradeCommand struct {
	UserID               string
	PartnerID            string
	Offer                []ItemChange
	Request              []ItemChange
	RequiresConfirmation bool
}
