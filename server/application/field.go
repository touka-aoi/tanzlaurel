package application

import (
	"context"
	"log/slog"

	"withered/server/domain"
)

// Field はマップとアクターを管理する構造体です。
type Field struct {
	Map    *Map
	Actors map[domain.SessionID]*Actor
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

// SpawnAtCenter はマップ中央にアクターを生成します。
func (f *Field) SpawnAtCenter(sessionID domain.SessionID) *Actor {
	actor := &Actor{
		SessionID: sessionID,
		Position: domain.Position2D{
			X: f.Map.WorldWidth() / 2,
			Y: f.Map.WorldHeight() / 2,
		},
		HP:    100,
		State: StateAlive,
	}
	f.Actors[sessionID] = actor
	return actor
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
			actor.Position.X = f.Map.WorldWidth() / 2
			actor.Position.Y = f.Map.WorldHeight() / 2
		}
	}
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
