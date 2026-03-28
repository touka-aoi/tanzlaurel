package application

import (
	"context"
	"math"
)

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

// --- RespawnSystem ---

type RespawnSystem struct{}

func (RespawnSystem) ID() string      { return "respawn" }
func (RespawnSystem) Phase() Phase    { return PhaseSimulation }
func (RespawnSystem) After() []string { return nil }
func (RespawnSystem) Run(_ context.Context, w *ShootingWorld) {
	for id := range w.Entities {
		ls, ok := w.LifeState[id]
		if !ok || ls.State&StateRespawning == 0 {
			continue
		}
		w.RespawnTimer[id]--
		if w.RespawnTimer[id] <= 0 {
			w.Health[id] = Health{HP: 100}
			w.LifeState[id] = LifeState{State: (ls.State &^ 0x0F) | StateAlive}
			w.Position[id] = w.randomPosition()
			delete(w.RespawnTimer, id)
		}
	}
}

// --- InputMoveSystem ---

type InputMoveSystem struct{}

func (InputMoveSystem) ID() string      { return "input_move" }
func (InputMoveSystem) Phase() Phase    { return PhaseSimulation }
func (InputMoveSystem) After() []string { return []string{"respawn"} }
func (InputMoveSystem) Run(_ context.Context, w *ShootingWorld) {
	mapW := w.Static.Map.WorldWidth()
	mapH := w.Static.Map.WorldHeight()

	for _, entry := range w.PendingInputs.Drain() {
		if _, ok := w.Entities[entry.EntityID]; !ok {
			continue
		}
		ls, ok := w.LifeState[entry.EntityID]
		if !ok || ls.State&StateAlive == 0 {
			continue
		}
		dx, dy := keyMaskToDirection(entry.KeyMask)
		if dx == 0 && dy == 0 {
			continue
		}
		pos := w.Position[entry.EntityID]
		pos.X = clamp(pos.X+dx*PlayerSpeed, 0, mapW)
		pos.Y = clamp(pos.Y+dy*PlayerSpeed, 0, mapH)
		w.Position[entry.EntityID] = pos
	}
}

// --- AutoShootSystem ---

type AutoShootSystem struct{}

func (AutoShootSystem) ID() string      { return "auto_shoot" }
func (AutoShootSystem) Phase() Phase    { return PhaseSimulation }
func (AutoShootSystem) After() []string { return []string{"input_move"} }
func (AutoShootSystem) Run(_ context.Context, w *ShootingWorld) {
	for id := range w.Entities {
		tag := w.Tag[id]
		if tag != TagPlayer && tag != TagBot {
			continue
		}
		ls, ok := w.LifeState[id]
		if !ok || ls.State&StateAlive == 0 {
			continue
		}

		w.ShootCooldown[id]--
		if w.ShootCooldown[id] > 0 {
			continue
		}

		pos := w.Position[id]

		// 最寄りの生存敵を探す
		var nearestID EntityID
		var nearestDistSq float32 = math.MaxFloat32
		found := false
		for otherID := range w.Entities {
			if otherID == id {
				continue
			}
			otherTag := w.Tag[otherID]
			if otherTag != TagPlayer && otherTag != TagBot {
				continue
			}
			otherLS, ok := w.LifeState[otherID]
			if !ok || otherLS.State&StateAlive == 0 {
				continue
			}
			otherPos := w.Position[otherID]
			dx := otherPos.X - pos.X
			dy := otherPos.Y - pos.Y
			distSq := dx*dx + dy*dy
			if distSq < nearestDistSq {
				nearestDistSq = distSq
				nearestID = otherID
				found = true
			}
		}
		if !found {
			continue
		}

		nearestPos := w.Position[nearestID]
		dx := nearestPos.X - pos.X
		dy := nearestPos.Y - pos.Y
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if dist < 0.001 {
			continue
		}

		vel := Velocity{
			X: dx / dist * BulletSpeed,
			Y: dy / dist * BulletSpeed,
		}
		w.SpawnBullet(id, pos, vel)
		w.ShootCooldown[id] = ShootCooldown
	}
}

// --- BulletMoveSystem ---

type BulletMoveSystem struct{}

func (BulletMoveSystem) ID() string      { return "bullet_move" }
func (BulletMoveSystem) Phase() Phase    { return PhaseSimulation }
func (BulletMoveSystem) After() []string { return []string{"auto_shoot"} }
func (BulletMoveSystem) Run(_ context.Context, w *ShootingWorld) {
	for id := range w.Entities {
		if w.Tag[id] != TagBullet {
			continue
		}
		pos := w.Position[id]
		vel := w.Velocity[id]
		pos.X += vel.X
		pos.Y += vel.Y
		w.Position[id] = pos
		w.TTL[id]--
	}
}

// --- CollisionDamageSystem ---

type CollisionDamageSystem struct{}

func (CollisionDamageSystem) ID() string      { return "collision_damage" }
func (CollisionDamageSystem) Phase() Phase    { return PhaseSimulation }
func (CollisionDamageSystem) After() []string { return []string{"bullet_move"} }
func (CollisionDamageSystem) Run(_ context.Context, w *ShootingWorld) {
	hitRadius := BulletRadius + ActorRadius
	hitRadiusSq := hitRadius * hitRadius

	// 命中弾丸を収集（イテレーション中の削除を避ける）
	var bulletsToRemove []EntityID

	for bulletID := range w.Entities {
		if w.Tag[bulletID] != TagBullet {
			continue
		}
		bulletPos := w.Position[bulletID]
		ownerID := w.Owner[bulletID]

		for actorID := range w.Entities {
			actorTag := w.Tag[actorID]
			if actorTag != TagPlayer && actorTag != TagBot {
				continue
			}
			if actorID == ownerID {
				continue
			}
			ls, ok := w.LifeState[actorID]
			if !ok || ls.State&StateAlive == 0 {
				continue
			}
			actorPos := w.Position[actorID]
			dx := bulletPos.X - actorPos.X
			dy := bulletPos.Y - actorPos.Y
			if dx*dx+dy*dy <= hitRadiusSq {
				// ダメージ適用
				hp := w.Health[actorID]
				if BulletDamage >= hp.HP {
					hp.HP = 0
					w.Health[actorID] = hp
					w.LifeState[actorID] = LifeState{State: (ls.State &^ 0x0F) | StateRespawning}
					w.RespawnTimer[actorID] = RespawnTicks
				} else {
					hp.HP -= BulletDamage
					w.Health[actorID] = hp
				}
				bulletsToRemove = append(bulletsToRemove, bulletID)
				break
			}
		}
	}

	for _, id := range bulletsToRemove {
		w.RemoveEntity(id)
	}
}

// --- BulletCleanupSystem ---

type BulletCleanupSystem struct{}

func (BulletCleanupSystem) ID() string      { return "bullet_cleanup" }
func (BulletCleanupSystem) Phase() Phase    { return PhaseSimulation }
func (BulletCleanupSystem) After() []string { return []string{"collision_damage"} }
func (BulletCleanupSystem) Run(_ context.Context, w *ShootingWorld) {
	var toRemove []EntityID
	for id := range w.Entities {
		if w.Tag[id] != TagBullet {
			continue
		}
		if w.TTL[id] == 0 {
			toRemove = append(toRemove, id)
		}
	}
	for _, id := range toRemove {
		w.RemoveEntity(id)
	}
}

// --- EncodeBroadcastSystem ---

type EncodeBroadcastSystem struct{}

func (EncodeBroadcastSystem) ID() string      { return "encode_broadcast" }
func (EncodeBroadcastSystem) Phase() Phase    { return PhaseNetwork }
func (EncodeBroadcastSystem) After() []string { return nil }
func (EncodeBroadcastSystem) Run(_ context.Context, _ *ShootingWorld) {
	// エンコードはTick()内でcodec.EncodeSnapshotが呼ばれるため、ここでは何もしない
}
