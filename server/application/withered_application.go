package application

import (
	"context"
	"log/slog"

	"withered/server/application/protocol"
	"withered/server/domain"
)

// WitheredApplication は各メッセージタイプを処理するApplication
type WitheredApplication struct {
	world         *ShootingWorld
	static        *Static
	scheduler     *Scheduler[*ShootingWorld]
	codec         Codec[*ShootingWorld]
	pendingInputs *PendingInput
}

func NewWitheredApplication() *WitheredApplication {
	gameMap := NewMap(100, 100, 1.0)
	field := NewField(gameMap)

	phases := []Phase{PhaseSimulation, PhaseNetwork}
	scheduler, err := NewScheduler[*ShootingWorld](phases,
		RespawnSystem{},
		InputMoveSystem{},
		AutoShootSystem{},
		BulletMoveSystem{},
		CollisionDamageSystem{},
		BulletCleanupSystem{},
		EncodeBroadcastSystem{},
	)
	if err != nil {
		panic("failed to create scheduler: " + err.Error())
	}

	return &WitheredApplication{
		world: &ShootingWorld{
			Entities:  make(map[EntityID]struct{}),
			Position:  make(map[EntityID]Position),
			Health:    make(map[EntityID]Health),
			LifeState: make(map[EntityID]LifeState),
			Velocity:  make(map[EntityID]Velocity),
			TTL:       make(map[EntityID]uint16),
			Owner:     make(map[EntityID]EntityID),
			Tag:       make(map[EntityID]Tag),
		},
		static:        field,
		scheduler:     scheduler,
		codec:         WitheredCodec{},
		pendingInputs: &PendingInput{},
	}
}

func (app *WitheredApplication) OnInput(packet []byte) {
	app.pendingInputs.Push(packet)
}

func (app *WitheredApplication) Tick(ctx context.Context) ([]byte, error) {
	app.scheduler.RunTick(ctx, app.world)
	b, err := app.codec.EncodeSnapshot(ctx, app.world)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// キーマスク定数
const (
	KeyW uint32 = 0x01 // 上
	KeyA uint32 = 0x02 // 左
	KeyS uint32 = 0x04 // 下
	KeyD uint32 = 0x08 // 右
)

// プレイヤー移動速度
const PlayerSpeed float32 = 1.0

// HandleMessage はAppPayloadを受け取り、アプリケーション固有のプロトコルに従って処理する。
// data はRoom層がRoomHeaderを除去した後の純粋なAppPayload。
func (app *WitheredApplication) HandleMessage(ctx context.Context, sessionID domain.SessionID, data []byte) error {
	// PayloadHeaderをパース（アプリケーション固有プロトコル）
	payloadHeader, err := protocol.ParsePayloadHeader(data)
	if err != nil {
		return err
	}

	payload := data[protocol.PayloadHeaderSize:]
	switch payloadHeader.DataType {
	case protocol.DataTypeInput:
		return app.handleInput(ctx, sessionID, payload)
	case protocol.DataTypeActor2D:
		return app.handleActor2D(ctx, sessionID, payloadHeader.SubType, payload)
	default:
		slog.WarnContext(ctx, "unknown data type", "dataType", payloadHeader.DataType)
		return nil
	}
}

func (app *WitheredApplication) handleInput(ctx context.Context, sessionID domain.SessionID, data []byte) error {
	input, err := protocol.ParseInputPayload(data)
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "handleInput",
		"sessionID", sessionID,
		"keyMask", input.KeyMask,
	)

	app.pendingInputs.Push(data)

	return nil
}

// keyMaskToDirection はキーマスクからX,Y方向を計算します。
func keyMaskToDirection(keyMask uint32) (dx, dy float32) {
	if keyMask&KeyW != 0 {
		dy -= 1
	}
	if keyMask&KeyS != 0 {
		dy += 1
	}
	if keyMask&KeyA != 0 {
		dx -= 1
	}
	if keyMask&KeyD != 0 {
		dx += 1
	}
	return dx, dy
}

func (app *WitheredApplication) handleActor2D(ctx context.Context, sessionID domain.SessionID, subType uint8, data []byte) error {
	switch protocol.ActorSubType(subType) {
	case protocol.ActorSubTypeSpawn:
		spawn, err := protocol.ParseActor2DSpawn(data)
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "handleActor2D:spawn",
			"sessionID", sessionID,
			"position", spawn.Position,
		)
	case protocol.ActorSubTypeUpdate:
		update, err := protocol.ParseActor2DUpdate(data)
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "handleActor2D:update",
			"sessionID", sessionID,
			"position", update.Position,
		)
	case protocol.ActorSubTypeDespawn:
		slog.DebugContext(ctx, "handleActor2D:despawn",
			"sessionID", sessionID,
		)
	default:
		slog.WarnContext(ctx, "unknown actor2d subtype", "subType", subType)
	}

	return nil
}

// encodeActorBroadcastMessage はアクターデータにPayloadHeaderを付与してAppPayloadを構築する。
// Room層がRoomMessage(AppData)としてラップして送信する。
func encodeActorBroadcastMessage(payload []byte) []byte {
	payloadHeader := protocol.PayloadHeader{
		DataType: protocol.DataTypeActor2D,
		SubType:  uint8(protocol.ActorSubTypeUpdate),
	}

	data := make([]byte, protocol.PayloadHeaderSize+len(payload))
	copy(data[:protocol.PayloadHeaderSize], payloadHeader.Encode())
	copy(data[protocol.PayloadHeaderSize:], payload)
	return data
}
