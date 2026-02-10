package application

import (
	"context"
	"encoding/binary"
	"log/slog"
	"math"
	"time"

	"withered/server/domain"
)

var byteOrder = binary.LittleEndian

// キーマスク定数
const (
	KeyW uint32 = 0x01 // 上
	KeyA uint32 = 0x02 // 左
	KeyS uint32 = 0x04 // 下
	KeyD uint32 = 0x08 // 右
)

// プレイヤー移動速度
const PlayerSpeed float32 = 1.0

// WitheredApplication は各メッセージタイプを処理するApplication
type WitheredApplication struct {
	field         *Field
	pendingInputs []InputEvent
}

// InputEvent は1つの入力イベントを表す
type InputEvent struct {
	SessionID domain.SessionID
	Header    *domain.Header
	Input     *domain.InputPayload
}

func NewWitheredApplication() *WitheredApplication {
	// マップ設定（ハードコーディング）
	gameMap := NewMap(100, 100, 1.0) // 100x100タイル、タイルサイズ1.0
	field := NewField(gameMap)

	return &WitheredApplication{
		field:         field,
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
	case domain.DataTypeActor2D:
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

func (app *WitheredApplication) handleActor(ctx context.Context, sessionID domain.SessionID, header *domain.Header, subType uint8, data []byte) error {
	switch domain.ActorSubType(subType) {
	case domain.ActorSubTypeSpawn:
		spawn, err := domain.ParseActor3DSpawn(data)
		if err != nil {
			return err
		}
		slog.DebugContext(ctx, "handleActor:spawn",
			"sessionID", sessionID,
			"seq", header.Seq,
			"position", spawn.Position,
		)
	case domain.ActorSubTypeUpdate:
		update, err := domain.ParseActor3DUpdate(data)
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
		actor := app.field.SpawnAtCenter(sessionID)
		slog.DebugContext(ctx, "handleControl:join",
			"sessionID", sessionID,
			"position", actor.Position,
		)
	case domain.ControlSubTypeLeave:
		app.field.Remove(sessionID)
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
	// pendingInputsを走査して移動処理を適用
	for _, event := range app.pendingInputs {
		dx, dy := keyMaskToDirection(event.Input.KeyMask)
		if dx != 0 || dy != 0 {
			app.field.ActorMove(ctx, event.SessionID, dx*PlayerSpeed, dy*PlayerSpeed)
		}
	}
	app.pendingInputs = app.pendingInputs[:0]

	// 全アクターの位置をエンコードして返す
	actors := app.field.GetAllActors()
	if len(actors) == 0 {
		return nil
	}

	payload := encodeActorPositions(actors)
	return encodeActorBroadcastMessage(payload)
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

// encodeActorPositions は全アクターの位置をバイナリにエンコードします。
// フォーマット: [ActorCount(u16)] + [Actor1] + [Actor2] + ...
// Actor: [SessionID([16]byte)] + [X(f32)] + [Y(f32)] = 24 bytes/actor
func encodeActorPositions(actors []*Actor) []byte {
	const actorSize = 24 // [16]byte + f32 + f32
	buf := make([]byte, 2+len(actors)*actorSize)

	// ActorCount (u16)
	byteOrder.PutUint16(buf[0:2], uint16(len(actors)))

	// 各アクター
	offset := 2
	for _, actor := range actors {
		bytes := actor.SessionID.Bytes()
		copy(buf[offset:offset+16], bytes[:])
		byteOrder.PutUint32(buf[offset+16:offset+20], math.Float32bits(actor.Position.X))
		byteOrder.PutUint32(buf[offset+20:offset+24], math.Float32bits(actor.Position.Y))
		offset += actorSize
	}

	return buf
}
