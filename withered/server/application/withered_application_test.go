package application

import (
	"context"
	"testing"

	"withered/server/application/protocol"
	"withered/server/domain"
)

func TestWitheredApplication_HandleMessage_Input(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()
	sessionID := domain.NewSessionID()

	// AppPayload = PayloadHeader + InputPayload
	payloadHeader := &protocol.PayloadHeader{
		DataType: protocol.DataTypeInput,
		SubType:  0,
	}
	input := &protocol.InputPayload{
		KeyMask: 0b1010,
	}

	data := append(payloadHeader.Encode(), input.Encode()...)

	err := app.HandleMessage(ctx, sessionID, data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestWitheredApplication_HandleMessage_Actor2DSpawn(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()
	sessionID := domain.NewSessionID()

	spawn := &protocol.Actor2DSpawn{
		Position: protocol.Position2D{X: 1.0, Y: 2.0},
	}
	payloadHeader := &protocol.PayloadHeader{
		DataType: protocol.DataTypeActor2D,
		SubType:  uint8(protocol.ActorSubTypeSpawn),
	}

	data := append(payloadHeader.Encode(), spawn.Encode()...)

	err := app.HandleMessage(ctx, sessionID, data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestWitheredApplication_HandleMessage_InvalidPayload(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()
	sessionID := domain.NewSessionID()

	// PayloadHeaderより短いデータ
	data := []byte{0x01}

	err := app.HandleMessage(ctx, sessionID, data)
	if err == nil {
		t.Fatal("expected error for invalid payload header")
	}
}

func TestWitheredApplication_Tick_Empty(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()

	result, err := app.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick failed: %v", err)
	}
	if result != nil {
		t.Error("expected nil result from empty world Tick")
	}
}

func TestWitheredApplication_Tick_WithEntity(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()

	sessionID := domain.NewSessionID()
	app.OnJoin(ctx, sessionID)

	result, err := app.Tick(ctx)
	if err != nil {
		t.Fatalf("Tick failed: %v", err)
	}
	if result == nil {
		t.Error("expected non-nil result from Tick with entity")
	}
}
