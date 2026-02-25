package application

type EntityID uint16
type Tag uint8

const (
	TagPlayer Tag = 1
	TagBot    Tag = 2
	TagBullet Tag = 3
)

// World はゲームの全体的な状態を管理する構造体です。
type ShootingWorld struct {
	Entities  map[EntityID]struct{}
	Position  map[EntityID]Position
	Health    map[EntityID]Health
	LifeState map[EntityID]LifeState
	Velocity  map[EntityID]Velocity
	TTL       map[EntityID]uint16
	Owner     map[EntityID]EntityID
	Tag       map[EntityID]Tag
}

func (w *ShootingWorld) RemoveEntity(id EntityID) {
	//TODO: 静的解析ですべてのコンポーネントを削除しているか保証したい
	delete(w.Entities, id)
	delete(w.Position, id)
	delete(w.Health, id)
	delete(w.LifeState, id)
	delete(w.Velocity, id)
	delete(w.TTL, id)
	delete(w.Owner, id)
	delete(w.Tag, id)
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
