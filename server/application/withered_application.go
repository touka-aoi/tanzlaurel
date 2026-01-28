package application

import (
	"context"
	"log/slog"

	"withered/server/domain"
)

// WitheredApplication は各メッセージタイプを処理するApplication
type WitheredApplication struct {
	pendingInputs []InputEvent
}

// InputEvent は1つの入力イベントを表す
type InputEvent struct {
	SessionID domain.SessionID
	Header    *domain.Header
	Input     *domain.InputPayload
}

func NewWitheredApplication() *WitheredApplication {
	return &WitheredApplication{
		pendingInputs: make([]InputEvent, 0),
	}
}

func (app *WitheredApplication) HandleMessage(ctx context.Context, sessionID domain.SessionID, data []byte) error {
	// 1. Headerをパース
	header, err := domain.ParseHeader(data)
	if err != nil {
		return err
	}

	// 2. PayloadHeaderをパース
	payloadData := data[domain.HeaderSize:]
	payloadHeader, err := domain.ParsePayloadHeader(payloadData)
	if err != nil {
		return err
	}

	// 3. 各DataTypeごとに処理
	payload := payloadData[domain.PayloadHeaderSize:]
	switch payloadHeader.DataType {
	case domain.DataTypeInput:
		return app.handleInput(ctx, sessionID, header, payload)
	case domain.DataTypeActor:
		return app.handleActor(ctx, sessionID, header, payloadHeader.SubType, payload)
	case domain.DataTypeVoice:
		return app.handleVoice(ctx, sessionID, header, payload)
	case domain.DataTypeControl:
		return app.handleControl(ctx, sessionID, header, payloadHeader.SubType, payload)
	default:
		slog.WarnContext(ctx, "unknown data type", "dataType", payloadHeader.DataType)
		return nil
	}
}

func (app *WitheredApplication) handleInput(ctx context.Context, sessionID domain.SessionID, header *domain.Header, data []byte) error {
	input, err := domain.ParseInputPayload(data)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "handleInput",
		"sessionID", sessionID,
		"seq", header.Seq,
		"keyMask", input.KeyMask,
	)

	app.pendingInputs = append(app.pendingInputs, InputEvent{
		SessionID: sessionID,
		Header:    header,
		Input:     input,
	})

	return nil
}

func (app *WitheredApplication) handleActor(ctx context.Context, sessionID domain.SessionID, header *domain.Header, subType uint8, data []byte) error {
	switch domain.ActorSubType(subType) {
	case domain.ActorSubTypeSpawn:
		spawn, err := domain.ParseActorSpawn(data)
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "handleActor:spawn",
			"sessionID", sessionID,
			"seq", header.Seq,
			"position", spawn.Position,
		)
	case domain.ActorSubTypeUpdate:
		update, err := domain.ParseActorUpdate(data)
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "handleActor:update",
			"sessionID", sessionID,
			"seq", header.Seq,
			"boneCount", len(update.Bones),
		)
	case domain.ActorSubTypeDespawn:
		slog.DebugContext(ctx, "handleActor:despawn",
			"sessionID", sessionID,
			"seq", header.Seq,
		)
	default:
		slog.WarnContext(ctx, "unknown actor subtype", "subType", subType)
	}

	return nil
}

func (app *WitheredApplication) handleVoice(ctx context.Context, sessionID domain.SessionID, header *domain.Header, data []byte) error {
	slog.DebugContext(ctx, "handleVoice",
		"sessionID", sessionID,
		"seq", header.Seq,
		"dataLen", len(data),
	)

	return nil
}

func (app *WitheredApplication) handleControl(ctx context.Context, sessionID domain.SessionID, header *domain.Header, subType uint8, data []byte) error {
	switch domain.ControlSubType(subType) {
	case domain.ControlSubTypeJoin:
		slog.DebugContext(ctx, "handleControl:join", "sessionID", sessionID)
	case domain.ControlSubTypeLeave:
		slog.DebugContext(ctx, "handleControl:leave", "sessionID", sessionID)
	case domain.ControlSubTypeKick:
		slog.DebugContext(ctx, "handleControl:kick", "sessionID", sessionID)
	case domain.ControlSubTypePing:
		slog.DebugContext(ctx, "handleControl:ping", "sessionID", sessionID)
	case domain.ControlSubTypePong:
		slog.DebugContext(ctx, "handleControl:pong", "sessionID", sessionID)
	case domain.ControlSubTypeError:
		slog.DebugContext(ctx, "handleControl:error", "sessionID", sessionID)
	default:
		slog.WarnContext(ctx, "unknown control subtype", "subType", subType)
	}

	return nil
}

func (app *WitheredApplication) Tick(ctx context.Context) interface{} {
	if len(app.pendingInputs) == 0 {
		return nil
	}

	app.pendingInputs = app.pendingInputs[:0]

	return nil
}
