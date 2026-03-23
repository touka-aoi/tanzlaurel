package application

import (
	"math/rand/v2"

	"withered/server/domain"
)

type (
	EntityID uint16
	Tag      uint8
)

const (
	TagPlayer Tag = 1
	TagBot    Tag = 2
	TagBullet Tag = 3
)

const spawnMargin float32 = 5.0

// ShootingWorld はゲームの全体的な状態を管理する構造体です。
type ShootingWorld struct {
	// コンポーネント
	Entities  map[EntityID]struct{}
	Position  map[EntityID]Position
	Health    map[EntityID]Health
	LifeState map[EntityID]LifeState
	Velocity  map[EntityID]Velocity
	TTL       map[EntityID]uint16
	Owner     map[EntityID]EntityID
	Tag       map[EntityID]Tag

	ShootCooldown map[EntityID]int
	RespawnTimer  map[EntityID]int

	// リソース
	SessionToEntity map[domain.SessionID]EntityID
	EntityToSession map[EntityID]domain.SessionID
	PendingInputs   *PendingInput
	Static          *Static
	NextEntityID    EntityID
}

func NewShootingWorld(static *Static) *ShootingWorld {
	return &ShootingWorld{
		Entities:        make(map[EntityID]struct{}),
		Position:        make(map[EntityID]Position),
		Health:          make(map[EntityID]Health),
		LifeState:       make(map[EntityID]LifeState),
		Velocity:        make(map[EntityID]Velocity),
		TTL:             make(map[EntityID]uint16),
		Owner:           make(map[EntityID]EntityID),
		Tag:             make(map[EntityID]Tag),
		ShootCooldown:   make(map[EntityID]int),
		RespawnTimer:    make(map[EntityID]int),
		SessionToEntity: make(map[domain.SessionID]EntityID),
		EntityToSession: make(map[EntityID]domain.SessionID),
		PendingInputs:   &PendingInput{},
		Static:          static,
	}
}

func (w *ShootingWorld) AllocEntity() EntityID {
	w.NextEntityID++
	id := w.NextEntityID
	w.Entities[id] = struct{}{}
	return id
}

func (w *ShootingWorld) SpawnActor(sessionID domain.SessionID, tag Tag) EntityID {
	id := w.AllocEntity()
	pos := w.randomPosition()
	w.Position[id] = pos
	w.Health[id] = Health{HP: 100}
	w.LifeState[id] = LifeState{State: StateAlive}
	w.Tag[id] = tag
	w.ShootCooldown[id] = 0
	w.SessionToEntity[sessionID] = id
	w.EntityToSession[id] = sessionID
	return id
}

func (w *ShootingWorld) SpawnBullet(ownerID EntityID, pos Position, vel Velocity) EntityID {
	id := w.AllocEntity()
	w.Position[id] = pos
	w.Velocity[id] = vel
	w.TTL[id] = BulletTTL
	w.Owner[id] = ownerID
	w.Tag[id] = TagBullet
	return id
}

func (w *ShootingWorld) RemoveEntity(id EntityID) {
	delete(w.Entities, id)
	delete(w.Position, id)
	delete(w.Health, id)
	delete(w.LifeState, id)
	delete(w.Velocity, id)
	delete(w.TTL, id)
	delete(w.Owner, id)
	delete(w.Tag, id)
	delete(w.ShootCooldown, id)
	delete(w.RespawnTimer, id)
	if sid, ok := w.EntityToSession[id]; ok {
		delete(w.SessionToEntity, sid)
		delete(w.EntityToSession, id)
	}
}

func (w *ShootingWorld) randomPosition() Position {
	m := w.Static.Map
	ww := m.WorldWidth() - spawnMargin*2
	wh := m.WorldHeight() - spawnMargin*2
	return Position{
		X: spawnMargin + rand.Float32()*ww,
		Y: spawnMargin + rand.Float32()*wh,
	}
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

type Position struct {
	X float32
	Y float32
}

type Health struct {
	HP uint8
}

type LifeState struct {
	State ActorState
}

type Velocity struct {
	X float32
	Y float32
}
