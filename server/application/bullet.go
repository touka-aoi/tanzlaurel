package application

import "withered/server/domain"

const (
	BulletSpeed   float32 = 1.5
	BulletTTL             = 120 // 2秒 @60FPS
	BulletRadius  float32 = 0.3
	ActorRadius   float32 = 0.5
	ShootCooldown         = 30 // 0.5秒 @60FPS
	BulletDamage  uint8   = 20
)

// Bullet はフィールド上の弾丸を表す構造体です。
type Bullet struct {
	ID       uint16
	OwnerID  domain.SessionID
	Position domain.Position2D
	Velocity domain.Position2D // VX, VY
	TTL      int
}

// HitEvent は弾丸がアクターに命中したイベントを表します。
type HitEvent struct {
	BulletID   uint16
	VictimID   domain.SessionID
	AttackerID domain.SessionID
}
