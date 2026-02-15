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
		actor := app.field.Spawn(sessionID)
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
	// 0. リスポーン処理
	app.field.TickRespawns()

	// 1. 人間入力適用 (移動)
	for _, event := range app.pendingInputs {
		dx, dy := keyMaskToDirection(event.Input.KeyMask)
		if dx != 0 || dy != 0 {
			app.field.ActorMove(ctx, event.SessionID, dx*PlayerSpeed, dy*PlayerSpeed)
		}
	}
	app.pendingInputs = app.pendingInputs[:0]

	// 2. 自動射撃
	app.processAutoShoot()

	// 3. 弾丸移動
	app.field.TickBullets()

	// 4. 衝突判定 → ダメージ適用
	hits := app.field.CheckBulletCollisions()
	for _, hit := range hits {
		app.field.DamageActor(hit.VictimID, BulletDamage)
	}

	// 5. 期限切れ弾丸除去
	app.field.RemoveExpiredBullets()

	// 6. エンコード → ブロードキャスト
	actors := app.field.GetAllActors()
	if len(actors) == 0 && len(app.field.Bullets) == 0 {
		return nil
	}

	payload := encodeGameState(actors, app.field.Bullets)
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

// processAutoShoot は全生存アクターが最寄り敵に向けて自動射撃を行います。
func (app *WitheredApplication) processAutoShoot() {
	actors := app.field.GetAllActors()
	for _, actor := range actors {
		if !actor.IsAlive() {
			continue
		}
		actor.ShootCooldown--
		if actor.ShootCooldown > 0 {
			continue
		}

		// 最寄りの生存敵を探す
		var nearest *Actor
		var nearestDistSq float32 = math.MaxFloat32
		for _, other := range actors {
			if other.SessionID == actor.SessionID || !other.IsAlive() {
				continue
			}
			dx := other.Position.X - actor.Position.X
			dy := other.Position.Y - actor.Position.Y
			distSq := dx*dx + dy*dy
			if distSq < nearestDistSq {
				nearestDistSq = distSq
				nearest = other
			}
		}
		if nearest == nil {
			continue
		}

		// 方向ベクトルを正規化して弾速を掛ける
		dx := nearest.Position.X - actor.Position.X
		dy := nearest.Position.Y - actor.Position.Y
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		if dist < 0.001 {
			continue
		}
		vx := dx / dist * BulletSpeed
		vy := dy / dist * BulletSpeed

		app.field.AddBullet(actor.SessionID, actor.Position, domain.Position2D{X: vx, Y: vy})
		actor.ShootCooldown = ShootCooldown
	}
}

// encodeGameState はアクターと弾丸を統合エンコードします。
// フォーマット: [ActorCount(u16)][Actors...][BulletCount(u16)][Bullets...]
func encodeGameState(actors []*Actor, bullets []*Bullet) []byte {
	const actorSize = 26  // [16]byte + f32 + f32 + u8 + u8
	const bulletSize = 34 // u16 + [16]byte + f32 + f32 + f32 + f32

	actorPayload := 2 + len(actors)*actorSize
	bulletPayload := 2 + len(bullets)*bulletSize
	buf := make([]byte, actorPayload+bulletPayload)

	// アクター部
	byteOrder.PutUint16(buf[0:2], uint16(len(actors)))
	offset := 2
	for _, actor := range actors {
		bytes := actor.SessionID.Bytes()
		copy(buf[offset:offset+16], bytes[:])
		byteOrder.PutUint32(buf[offset+16:offset+20], math.Float32bits(actor.Position.X))
		byteOrder.PutUint32(buf[offset+20:offset+24], math.Float32bits(actor.Position.Y))
		buf[offset+24] = actor.HP
		buf[offset+25] = uint8(actor.State)
		offset += actorSize
	}

	// 弾丸部
	byteOrder.PutUint16(buf[offset:offset+2], uint16(len(bullets)))
	offset += 2
	for _, b := range bullets {
		byteOrder.PutUint16(buf[offset:offset+2], b.ID)
		ownerBytes := b.OwnerID.Bytes()
		copy(buf[offset+2:offset+18], ownerBytes[:])
		byteOrder.PutUint32(buf[offset+18:offset+22], math.Float32bits(b.Position.X))
		byteOrder.PutUint32(buf[offset+22:offset+26], math.Float32bits(b.Position.Y))
		byteOrder.PutUint32(buf[offset+26:offset+30], math.Float32bits(b.Velocity.X))
		byteOrder.PutUint32(buf[offset+30:offset+34], math.Float32bits(b.Velocity.Y))
		offset += bulletSize
	}

	return buf
}

// encodeActorPositions は全アクターの位置・HP・状態をバイナリにエンコードします。
// フォーマット: [ActorCount(u16)] + [Actor1] + [Actor2] + ...
// Actor: [SessionID([16]byte)] + [X(f32)] + [Y(f32)] + [HP(u8)] + [State(u8)] = 26 bytes/actor
func encodeActorPositions(actors []*Actor) []byte {
	const actorSize = 26 // [16]byte + f32 + f32 + u8 + u8
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
		buf[offset+24] = actor.HP
		buf[offset+25] = uint8(actor.State)
		offset += actorSize
	}

	return buf
}
