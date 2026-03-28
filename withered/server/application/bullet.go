package application

const (
	BulletSpeed   float32 = 1.5
	BulletTTL             = 120 // 2秒 @60FPS
	BulletRadius  float32 = 0.3
	ActorRadius   float32 = 0.5
	ShootCooldown         = 30 // 0.5秒 @60FPS
	BulletDamage  uint8   = 20
	RespawnTicks          = 180 // 3秒 @60FPS
)
