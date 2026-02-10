package application

import (
	"context"
	"testing"

	"withered/server/domain"
)

func TestWitheredApplication_HandleMessage_Input(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()
	sessionID := domain.NewSessionID()

	// Inputメッセージを作成
	sessionBytes := sessionID.Bytes()
	header := &domain.Header{
		Version:   1,
		SessionID: sessionBytes,
		Seq:       1,
		Length:    domain.PayloadHeaderSize + domain.InputPayloadSize,
		Timestamp: 1000,
	}
	payloadHeader := &domain.PayloadHeader{
		DataType: domain.DataTypeInput,
		SubType:  0,
	}
	input := &domain.InputPayload{
		KeyMask: 0b1010,
	}

	data := append(header.Encode(), payloadHeader.Encode()...)
	data = append(data, input.Encode()...)

	err := app.HandleMessage(ctx, sessionID, data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestWitheredApplication_HandleMessage_ActorSpawn(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()
	sessionID := domain.NewSessionID()

	sessionBytes := sessionID.Bytes()
	header := &domain.Header{
		Version:   1,
		SessionID: sessionBytes,
		Seq:       1,
		Length:    domain.PayloadHeaderSize + domain.PositionSize,
		Timestamp: 1000,
	}
	payloadHeader := &domain.PayloadHeader{
		DataType: domain.DataTypeActor2D,
		SubType:  uint8(domain.ActorSubTypeSpawn),
	}
	spawn := &domain.ActorSpawn{
		Position: domain.Position{X: 1.0, Y: 2.0, Z: 3.0, QX: 0, QY: 0, QZ: 0, QW: 1},
	}

	data := append(header.Encode(), payloadHeader.Encode()...)
	data = append(data, spawn.Encode()...)

	err := app.HandleMessage(ctx, sessionID, data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestWitheredApplication_HandleMessage_Control(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()
	sessionID := domain.NewSessionID()

	sessionBytes := sessionID.Bytes()
	header := &domain.Header{
		Version:   1,
		SessionID: sessionBytes,
		Seq:       1,
		Length:    domain.PayloadHeaderSize,
		Timestamp: 1000,
	}
	payloadHeader := &domain.PayloadHeader{
		DataType: domain.DataTypeControl,
		SubType:  uint8(domain.ControlSubTypeJoin),
	}

	data := append(header.Encode(), payloadHeader.Encode()...)

	err := app.HandleMessage(ctx, sessionID, data)
	if err != nil {
		t.Fatalf("HandleMessage failed: %v", err)
	}
}

func TestWitheredApplication_HandleMessage_InvalidHeader(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()
	sessionID := domain.NewSessionID()

	// 短すぎるデータ
	data := []byte{0x01, 0x02}

	err := app.HandleMessage(ctx, sessionID, data)
	if err == nil {
		t.Fatal("expected error for invalid header")
	}
}

func TestWitheredApplication_Tick(t *testing.T) {
	app := NewWitheredApplication()
	ctx := context.Background()

	// Tickはnilを返す
	result := app.Tick(ctx)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}
