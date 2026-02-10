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

// Actor はフィールド上のプレイヤーを表す構造体です。
type Actor struct {
	SessionID domain.SessionID
	Position  domain.Position2D
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

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
