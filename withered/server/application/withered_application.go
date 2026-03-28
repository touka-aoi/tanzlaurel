package application

import (
	"context"
	"log/slog"

	"withered/server/application/protocol"
	"withered/server/domain"
)

// WitheredApplication は各メッセージタイプを処理するApplication
type WitheredApplication struct {
	world     *ShootingWorld
	scheduler *Scheduler[*ShootingWorld]
	codec     Codec[*ShootingWorld]
}

func NewWitheredApplication() *WitheredApplication {
	gameMap := NewMap(100, 100, 1.0)
	static := NewField(gameMap)

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
		world:     NewShootingWorld(static),
		scheduler: scheduler,
		codec:     WitheredCodec{},
	}
}

// OnJoin はセッション参加時にプレイヤーEntityを生成する。
func (app *WitheredApplication) OnJoin(ctx context.Context, sessionID domain.SessionID) {
	entityID := app.world.SpawnActor(sessionID, TagPlayer)
	slog.InfoContext(ctx, "player entity spawned",
		"sessionID", sessionID,
		"entityID", entityID,
	)
}

// OnLeave はセッション離脱時にプレイヤーEntityを削除する。
func (app *WitheredApplication) OnLeave(ctx context.Context, sessionID domain.SessionID) {
	entityID, ok := app.world.SessionToEntity[sessionID]
	if !ok {
		slog.WarnContext(ctx, "OnLeave: no entity for session", "sessionID", sessionID)
		return
	}
	app.world.RemoveEntity(entityID)
	slog.InfoContext(ctx, "player entity removed",
		"sessionID", sessionID,
		"entityID", entityID,
	)
}

func (app *WitheredApplication) Tick(ctx context.Context) ([]byte, error) {
	app.scheduler.RunTick(ctx, app.world)
	if len(app.world.Entities) == 0 {
		return nil, nil
	}
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
func (app *WitheredApplication) HandleMessage(ctx context.Context, sessionID domain.SessionID, data []byte) error {
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

	entityID, ok := app.world.SessionToEntity[sessionID]
	if !ok {
		slog.WarnContext(ctx, "handleInput: no entity for session", "sessionID", sessionID)
		return nil
	}

	app.world.PendingInputs.Push(InputEntry{
		EntityID: entityID,
		KeyMask:  input.KeyMask,
	})

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
