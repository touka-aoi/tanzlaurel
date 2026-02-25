package application

import "context"

type Phase uint8

const (
	PhaseSimulation Phase = iota
	PhaseNetwork
)

type System[W any] interface {
	ID() string
	Phase() Phase
	After() []string // 自分より先に終わっていて欲しいSystem IDs
	Run(ctx context.Context, world W)
}

// コンパイル時ガード: 各SystemがSystem[*ShootingWorld]を満たすことを保証
var (
	_ System[*ShootingWorld] = RespawnSystem{}
	_ System[*ShootingWorld] = InputMoveSystem{}
	_ System[*ShootingWorld] = AutoShootSystem{}
	_ System[*ShootingWorld] = BulletMoveSystem{}
	_ System[*ShootingWorld] = CollisionDamageSystem{}
	_ System[*ShootingWorld] = BulletCleanupSystem{}
	_ System[*ShootingWorld] = EncodeBroadcastSystem{}
)

type RespawnSystem struct{}

func (RespawnSystem) ID() string      { return "respawn" }
func (RespawnSystem) Phase() Phase    { return PhaseSimulation }
func (RespawnSystem) After() []string { return nil }
func (RespawnSystem) Run(ctx context.Context, world *ShootingWorld) {
	// TODO: リスポーンロジックを実装
}

type InputMoveSystem struct{}

func (InputMoveSystem) ID() string      { return "input_move" }
func (InputMoveSystem) Phase() Phase    { return PhaseSimulation }
func (InputMoveSystem) After() []string { return []string{"respawn"} }
func (InputMoveSystem) Run(ctx context.Context, world *ShootingWorld) {
	// TODO: 入力に基づく移動ロジックを実装
}

type AutoShootSystem struct{}

func (AutoShootSystem) ID() string      { return "auto_shoot" }
func (AutoShootSystem) Phase() Phase    { return PhaseSimulation }
func (AutoShootSystem) After() []string { return []string{"input_move"} }
func (AutoShootSystem) Run(ctx context.Context, world *ShootingWorld) {
	// TODO: 自動射撃ロジックを実装
}

type BulletMoveSystem struct{}

func (BulletMoveSystem) ID() string      { return "bullet_move" }
func (BulletMoveSystem) Phase() Phase    { return PhaseSimulation }
func (BulletMoveSystem) After() []string { return []string{"auto_shoot"} }
func (BulletMoveSystem) Run(ctx context.Context, world *ShootingWorld) {
	// TODO: 弾丸移動ロジックを実装
}

type CollisionDamageSystem struct{}

func (CollisionDamageSystem) ID() string      { return "collision_damage" }
func (CollisionDamageSystem) Phase() Phase    { return PhaseSimulation }
func (CollisionDamageSystem) After() []string { return []string{"bullet_move"} }
func (CollisionDamageSystem) Run(ctx context.Context, world *ShootingWorld) {
	// TODO: 衝突ダメージロジックを実装
}

type BulletCleanupSystem struct{}

func (BulletCleanupSystem) ID() string      { return "bullet_cleanup" }
func (BulletCleanupSystem) Phase() Phase    { return PhaseSimulation }
func (BulletCleanupSystem) After() []string { return []string{"collision_damage"} }
func (BulletCleanupSystem) Run(ctx context.Context, world *ShootingWorld) {
	// TODO: 弾丸クリーンアップロジックを実装
}

type EncodeBroadcastSystem struct{}

func (EncodeBroadcastSystem) ID() string      { return "encode_broadcast" }
func (EncodeBroadcastSystem) Phase() Phase    { return PhaseNetwork }
func (EncodeBroadcastSystem) After() []string { return nil }
func (EncodeBroadcastSystem) Run(ctx context.Context, world *ShootingWorld) {
	// TODO: ブロードキャストエンコードロジックを実装
}
