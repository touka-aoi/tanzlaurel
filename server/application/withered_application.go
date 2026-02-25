package application

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	"withered/server/domain"
)

// WitheredApplication は各メッセージタイプを処理するApplication
type WitheredApplication struct {
	world         *ShootingWorld
	static        *Static
	scheduler     *Scheduler[*ShootingWorld]
	codec         Codec[*ShootingWorld]
	pendingInputs *PendingInput
	nextEntityID  EntityID
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
		nextEntityID:  1,
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
	case domain.DataTypeActor2D:
		return app.handleActor2D(ctx, sessionID, header, payloadHeader.SubType, payload)
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

func (app *WitheredApplication) handleActor2D(ctx context.Context, sessionID domain.SessionID, header *domain.Header, subType uint8, data []byte) error {
	switch domain.ActorSubType(subType) {
	case domain.ActorSubTypeSpawn:
		spawn, err := domain.ParseActor2DSpawn(data)
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "handleActor2D:spawn",
			"sessionID", sessionID,
			"seq", header.Seq,
			"position", spawn.Position,
		)
	case domain.ActorSubTypeUpdate:
		update, err := domain.ParseActor2DUpdate(data)
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "handleActor2D:update",
			"sessionID", sessionID,
			"seq", header.Seq,
			"position", update.Position,
		)
	case domain.ActorSubTypeDespawn:
		slog.DebugContext(ctx, "handleActor2D:despawn",
			"sessionID", sessionID,
			"seq", header.Seq,
		)
	default:
		slog.WarnContext(ctx, "unknown actor2d subtype", "subType", subType)
	}

	return nil
}

func (app *WitheredApplication) handleControl(ctx context.Context, sessionID domain.SessionID, header *domain.Header, subType uint8, data []byte) error {
	switch domain.ControlSubType(subType) {
	case domain.ControlSubTypeJoin:
		id := app.spawnEntity(TagPlayer)
		w := app.static.Map.WorldWidth()
		h := app.static.Map.WorldHeight()
		const margin float32 = 5.0
		app.world.Position[id] = Position{
			X: margin + rand.Float32()*(w-margin*2),
			Y: margin + rand.Float32()*(h-margin*2),
		}
		app.world.Health[id] = Health{HP: 100}
		app.world.LifeState[id] = LifeState{State: StateAlive}
		slog.DebugContext(ctx, "handleControl:join",
			"sessionID", sessionID,
			"entityID", id,
		)
	case domain.ControlSubTypeLeave:
		// TODO: SessionID → EntityID のマッピングが必要
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

func (app *WitheredApplication) spawnEntity(tag Tag) EntityID {
	id := app.nextEntityID
	app.nextEntityID++
	app.world.Entities[id] = struct{}{}
	app.world.Tag[id] = tag
	return id
}

// encodeActorBroadcastMessage はアクターデータにHeader+PayloadHeaderを付与して完全なプロトコルメッセージを構築します。
func encodeActorBroadcastMessage(payload []byte) []byte {
	header := domain.Header{
		Version:   1,
		SessionID: [16]byte{}, // サーバー発のブロードキャスト
		Seq:       0,
		Length:    uint16(domain.PayloadHeaderSize + len(payload)),
		Timestamp: uint32(time.Now().UnixMilli() & 0xFFFFFFFF),
	}
	payloadHeader := domain.PayloadHeader{
		DataType: domain.DataTypeActor2D,
		SubType:  uint8(domain.ActorSubTypeUpdate),
	}

	data := make([]byte, domain.HeaderSize+domain.PayloadHeaderSize+len(payload))
	copy(data[:domain.HeaderSize], header.Encode())
	copy(data[domain.HeaderSize:domain.HeaderSize+domain.PayloadHeaderSize], payloadHeader.Encode())
	copy(data[domain.HeaderSize+domain.PayloadHeaderSize:], payload)
	return data
}
