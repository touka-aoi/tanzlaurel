package application

import (
	"context"
	"log/slog"
	"math/rand/v2"

	"withered/server/domain"
)

// Field はマップとアクターを管理する構造体です。
type Field struct {
	Map          *Map
	Actors       map[domain.SessionID]*Actor
	Bullets      []*Bullet
	nextBulletID uint16
}

// ActorState はアクターの状態と種別をビットマスクで表現します。
// bit 0-3: 状態フラグ, bit 4-7: 種別フラグ
type ActorState uint8

const (
	StateAlive      ActorState = 0x01
	StateRespawning ActorState = 0x02
	KindPlayer      ActorState = 0x00
	KindBot         ActorState = 0x10
)

// Actor はフィールド上のプレイヤーを表す構造体です。
type Actor struct {
	SessionID     domain.SessionID
	Position      domain.Position2D
	HP            uint8
	State         ActorState
	ShootCooldown int
	RespawnTimer  int
}

// NewField は指定されたマップでフィールドを作成します。
func NewField(m *Map) *Field {
	return &Field{
		Map:    m,
		Actors: make(map[domain.SessionID]*Actor),
	}
}

const spawnMargin float32 = 5.0 // 端から5ユニットの余白

// Spawn はマップ内のランダム位置にアクターを生成します。
func (f *Field) Spawn(sessionID domain.SessionID) *Actor {
	actor := &Actor{
		SessionID: sessionID,
		Position:  f.randomPosition(),
		HP:        100,
		State:     StateAlive,
	}
	f.Actors[sessionID] = actor
	return actor
}

func (f *Field) randomPosition() domain.Position2D {
	w := f.Map.WorldWidth() - spawnMargin*2
	h := f.Map.WorldHeight() - spawnMargin*2
	return domain.Position2D{
		X: spawnMargin + rand.Float32()*w,
		Y: spawnMargin + rand.Float32()*h,
	}
}

// ActorMove はアクターを移動させます。境界を超えないようにクランプします。
func (f *Field) ActorMove(ctx context.Context, sessionID domain.SessionID, dx, dy float32) {
	actor, ok := f.Actors[sessionID]
	if !ok {
		slog.WarnContext(ctx, "actor not found", "sessionID", sessionID)
		return
	}

	actor.Position.X = clamp(actor.Position.X+dx, 0, f.Map.WorldWidth())
	actor.Position.Y = clamp(actor.Position.Y+dy, 0, f.Map.WorldHeight())
}

// Remove はアクターをフィールドから削除します。
func (f *Field) Remove(sessionID domain.SessionID) {
	delete(f.Actors, sessionID)
}

// GetAllActors は全アクターのスライスを返します。
func (f *Field) GetAllActors() []*Actor {
	actors := make([]*Actor, 0, len(f.Actors))
	for _, actor := range f.Actors {
		actors = append(actors, actor)
	}
	return actors
}

// GetActor は指定されたセッションIDのアクターを取得します。
func (f *Field) GetActor(sessionID domain.SessionID) (*Actor, bool) {
	actor, ok := f.Actors[sessionID]
	return actor, ok
}

const RespawnTicks = 180 // 3秒 @60FPS

// IsAlive はアクターが生存しているかを返します。
func (a *Actor) IsAlive() bool {
	return a.State&StateAlive != 0
}

// DamageActor はアクターにダメージを与えます。HPが0になったらRespawning状態に遷移します。
func (f *Field) DamageActor(sessionID domain.SessionID, damage uint8) {
	actor, ok := f.Actors[sessionID]
	if !ok || !actor.IsAlive() {
		return
	}

	if damage >= actor.HP {
		actor.HP = 0
		actor.State = (actor.State &^ 0x0F) | StateRespawning // 状態フラグのみ変更、種別フラグは維持
		actor.RespawnTimer = RespawnTicks
	} else {
		actor.HP -= damage
	}
}

// TickRespawns はリスポーンタイマーを進め、復活処理を行います。
func (f *Field) TickRespawns() {
	for _, actor := range f.Actors {
		if actor.State&StateRespawning == 0 {
			continue
		}
		actor.RespawnTimer--
		if actor.RespawnTimer <= 0 {
			actor.HP = 100
			actor.State = (actor.State &^ 0x0F) | StateAlive
			pos := f.randomPosition()
			actor.Position = pos
		}
	}
}

// AddBullet は弾丸を生成してフィールドに追加します。
func (f *Field) AddBullet(ownerID domain.SessionID, position, velocity domain.Position2D) {
	f.nextBulletID++
	bullet := &Bullet{
		ID:       f.nextBulletID,
		OwnerID:  ownerID,
		Position: position,
		Velocity: velocity,
		TTL:      BulletTTL,
	}
	f.Bullets = append(f.Bullets, bullet)
}

// TickBullets は全弾丸の位置を更新し、TTLを減算します。
func (f *Field) TickBullets() {
	for _, b := range f.Bullets {
		b.Position.X += b.Velocity.X
		b.Position.Y += b.Velocity.Y
		b.TTL--
	}
}

// CheckBulletCollisions は弾丸とアクターの衝突を判定します。
// 命中した弾丸は除去され、HitEventのスライスを返します。
func (f *Field) CheckBulletCollisions() []HitEvent {
	var hits []HitEvent
	hitRadius := BulletRadius + ActorRadius
	hitRadiusSq := hitRadius * hitRadius

	surviving := f.Bullets[:0]
	for _, b := range f.Bullets {
		hit := false
		for _, actor := range f.Actors {
			if actor.SessionID == b.OwnerID || !actor.IsAlive() {
				continue
			}
			dx := b.Position.X - actor.Position.X
			dy := b.Position.Y - actor.Position.Y
			if dx*dx+dy*dy <= hitRadiusSq {
				hits = append(hits, HitEvent{
					BulletID:   b.ID,
					VictimID:   actor.SessionID,
					AttackerID: b.OwnerID,
				})
				hit = true
				break
			}
		}
		if !hit {
			surviving = append(surviving, b)
		}
	}
	f.Bullets = surviving
	return hits
}

// RemoveExpiredBullets はTTLが0以下の弾丸を除去します。
func (f *Field) RemoveExpiredBullets() {
	surviving := f.Bullets[:0]
	for _, b := range f.Bullets {
		if b.TTL > 0 {
			surviving = append(surviving, b)
		}
	}
	f.Bullets = surviving
}

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
