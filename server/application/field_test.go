package application

import (
	"context"
	"testing"

	"withered/server/domain"
)

func TestNewField(t *testing.T) {
	m := NewMap(10, 10, 1.0)
	f := NewField(m)

	if f.Map != m {
		t.Error("Map not set correctly")
	}
	if len(f.Actors) != 0 {
		t.Errorf("Actors length = %d, want 0", len(f.Actors))
	}
}

func TestField_Spawn(t *testing.T) {
	m := NewMap(10, 10, 1.0) // WorldWidth=10, WorldHeight=10
	f := NewField(m)

	sessionID := domain.NewSessionID()
	actor := f.Spawn(sessionID)

	if actor.SessionID != sessionID {
		t.Errorf("SessionID = %s, want %s", actor.SessionID, sessionID)
	}
	if actor.Position.X < 0 || actor.Position.X > 10.0 {
		t.Errorf("Position.X = %f, want within [0, 10]", actor.Position.X)
	}
	if actor.Position.Y < 0 || actor.Position.Y > 10.0 {
		t.Errorf("Position.Y = %f, want within [0, 10]", actor.Position.Y)
	}
	if actor.HP != 100 {
		t.Errorf("HP = %d, want 100", actor.HP)
	}
	if !actor.IsAlive() {
		t.Error("actor should be alive")
	}
	if len(f.Actors) != 1 {
		t.Errorf("Actors length = %d, want 1", len(f.Actors))
	}
}

func TestField_Move(t *testing.T) {
	m := NewMap(100, 100, 1.0)
	f := NewField(m)
	ctx := context.Background()

	sessionID := domain.NewSessionID()
	spawned := f.Spawn(sessionID)
	startX := spawned.Position.X
	startY := spawned.Position.Y

	f.ActorMove(ctx, sessionID, 2.0, -1.0)

	actor, ok := f.GetActor(sessionID)
	if !ok {
		t.Fatal("actor not found")
	}
	if actor.Position.X != startX+2.0 || actor.Position.Y != startY-1.0 {
		t.Errorf("Position = (%f, %f), want (%f, %f)", actor.Position.X, actor.Position.Y, startX+2.0, startY-1.0)
	}
}

func TestField_Move_Clamp(t *testing.T) {
	m := NewMap(10, 10, 1.0) // WorldWidth=10, WorldHeight=10
	f := NewField(m)
	ctx := context.Background()

	sessionID := domain.NewSessionID()
	f.Spawn(sessionID)

	// 大きく右に移動 → X=10にクランプ
	f.ActorMove(ctx, sessionID, 1000, 0)
	actor, _ := f.GetActor(sessionID)
	if actor.Position.X != 10.0 {
		t.Errorf("clamp max x: Position.X = %f, want 10.0", actor.Position.X)
	}

	// 大きく左に移動 → X=0にクランプ
	f.ActorMove(ctx, sessionID, -1000, 0)
	actor, _ = f.GetActor(sessionID)
	if actor.Position.X != 0.0 {
		t.Errorf("clamp min x: Position.X = %f, want 0.0", actor.Position.X)
	}

	// 大きく下に移動 → Y=10にクランプ
	f.ActorMove(ctx, sessionID, 0, 1000)
	actor, _ = f.GetActor(sessionID)
	if actor.Position.Y != 10.0 {
		t.Errorf("clamp max y: Position.Y = %f, want 10.0", actor.Position.Y)
	}

	// 大きく上に移動 → Y=0にクランプ
	f.ActorMove(ctx, sessionID, 0, -1000)
	actor, _ = f.GetActor(sessionID)
	if actor.Position.Y != 0.0 {
		t.Errorf("clamp min y: Position.Y = %f, want 0.0", actor.Position.Y)
	}
}

func TestField_Move_ActorNotFound(t *testing.T) {
	m := NewMap(10, 10, 1.0)
	f := NewField(m)
	ctx := context.Background()

	// 存在しないアクターへのMoveはパニックしない（警告ログのみ）
	nonExistentID := domain.NewSessionID()
	f.ActorMove(ctx, nonExistentID, 1.0, 1.0)
}

func TestField_Remove(t *testing.T) {
	m := NewMap(10, 10, 1.0)
	f := NewField(m)

	sessionID1 := domain.NewSessionID()
	sessionID2 := domain.NewSessionID()
	f.Spawn(sessionID1)
	f.Spawn(sessionID2)

	if len(f.Actors) != 2 {
		t.Fatalf("Actors length = %d, want 2", len(f.Actors))
	}

	f.Remove(sessionID1)

	if len(f.Actors) != 1 {
		t.Errorf("Actors length = %d, want 1", len(f.Actors))
	}
	if _, ok := f.GetActor(sessionID1); ok {
		t.Error("actor 1 should be removed")
	}
	if _, ok := f.GetActor(sessionID2); !ok {
		t.Error("actor 2 should exist")
	}
}

func TestField_GetAllActors(t *testing.T) {
	m := NewMap(10, 10, 1.0)
	f := NewField(m)

	f.Spawn(domain.NewSessionID())
	f.Spawn(domain.NewSessionID())
	f.Spawn(domain.NewSessionID())

	actors := f.GetAllActors()
	if len(actors) != 3 {
		t.Errorf("actors length = %d, want 3", len(actors))
	}
}
