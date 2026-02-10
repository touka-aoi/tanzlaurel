package domain

import (
	"encoding/binary"
	"errors"
	"math"
	"time"
)

// バイトオーダー: リトルエンディアン
var byteOrder = binary.LittleEndian

const (
	HeaderSize        = 25
	PayloadHeaderSize = 2
	JoinPayloadSize   = 16
)

// Header はメッセージヘッダー (25バイト)
//
//	version    u8      (1)
//	sessionID  [16]byte (16)
//	seq        u16     (2)
//	length     u16     (2)  - ペイロード長
//	timestamp  u32     (4)
type Header struct {
	Version   uint8
	SessionID [16]byte
	Seq       uint16
	Length    uint16
	Timestamp uint32
}

// DataType はメッセージの種別
type DataType uint8

const (
	DataTypeInput    DataType = 1
	DataTypeActor2D  DataType = 2
	DataTypeVoice    DataType = 3
	DataTypeControl  DataType = 4
	DataTypeActor3D  DataType = 5
)

// ActorSubType はactorメッセージのサブタイプ
type ActorSubType uint8

const (
	ActorSubTypeSpawn   ActorSubType = 1
	ActorSubTypeUpdate  ActorSubType = 2
	ActorSubTypeDespawn ActorSubType = 3
)

// ControlSubType はcontrolメッセージのサブタイプ
type ControlSubType uint8

const (
	ControlSubTypeJoin   ControlSubType = 1
	ControlSubTypeLeave  ControlSubType = 2
	ControlSubTypeKick   ControlSubType = 3
	ControlSubTypePing   ControlSubType = 4
	ControlSubTypePong   ControlSubType = 5
	ControlSubTypeError  ControlSubType = 6
	ControlSubTypeAssign ControlSubType = 7
)

// PayloadHeader はペイロードヘッダー (2バイト)
//
//	datatype  u8 (1)
//	subtype   u8 (1)
type PayloadHeader struct {
	DataType DataType
	SubType  uint8
}

var (
	ErrInvalidHeaderSize  = errors.New("invalid header size")
	ErrInvalidPayloadSize = errors.New("invalid payload size")
)

// ParseHeader はバイト列からHeaderをパースする
func ParseHeader(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		return nil, ErrInvalidHeaderSize
	}

	var sessionID [16]byte
	copy(sessionID[:], data[1:17])

	return &Header{
		Version:   data[0],
		SessionID: sessionID,
		Seq:       byteOrder.Uint16(data[17:19]),
		Length:    byteOrder.Uint16(data[19:21]),
		Timestamp: byteOrder.Uint32(data[21:25]),
	}, nil
}

// Encode はHeaderをバイト列にエンコードする
func (h *Header) Encode() []byte {
	data := make([]byte, HeaderSize)
	data[0] = h.Version
	copy(data[1:17], h.SessionID[:])
	byteOrder.PutUint16(data[17:19], h.Seq)
	byteOrder.PutUint16(data[19:21], h.Length)
	byteOrder.PutUint32(data[21:25], h.Timestamp)
	return data
}

// ParsePayloadHeader はバイト列からPayloadHeaderをパースする
func ParsePayloadHeader(data []byte) (*PayloadHeader, error) {
	if len(data) < PayloadHeaderSize {
		return nil, ErrInvalidPayloadSize
	}

	return &PayloadHeader{
		DataType: DataType(data[0]),
		SubType:  data[1],
	}, nil
}

// Encode はPayloadHeaderをバイト列にエンコードする
func (p *PayloadHeader) Encode() []byte {
	data := make([]byte, PayloadHeaderSize)
	data[0] = byte(p.DataType)
	data[1] = byte(p.SubType)
	return data
}

// EncodeAssignMessage はセッションID通知メッセージをエンコードする
// クライアントに自分のセッションIDを通知するために使用
func EncodeAssignMessage(sessionID SessionID) []byte {
	header := Header{
		Version:   1,
		SessionID: sessionID.Bytes(),
		Seq:       0,
		Length:    PayloadHeaderSize,
		Timestamp: uint32(time.Now().UnixMilli() & 0xFFFFFFFF),
	}
	payloadHeader := PayloadHeader{
		DataType: DataTypeControl,
		SubType:  uint8(ControlSubTypeAssign),
	}

	data := make([]byte, HeaderSize+PayloadHeaderSize)
	copy(data[:HeaderSize], header.Encode())
	copy(data[HeaderSize:], payloadHeader.Encode())
	return data
}

// EncodeLeaveMessage はルーム離脱メッセージをエンコードする
// 異常切断時にclose()からRoom離脱を通知するために使用
func EncodeLeaveMessage(sessionID SessionID) []byte {
	header := Header{
		Version:   1,
		SessionID: sessionID.Bytes(),
		Seq:       0,
		Length:    PayloadHeaderSize,
		Timestamp: uint32(time.Now().UnixMilli() & 0xFFFFFFFF),
	}
	payloadHeader := PayloadHeader{
		DataType: DataTypeControl,
		SubType:  uint8(ControlSubTypeLeave),
	}

	data := make([]byte, HeaderSize+PayloadHeaderSize)
	copy(data[:HeaderSize], header.Encode())
	copy(data[HeaderSize:], payloadHeader.Encode())
	return data
}

// EncodePingMessage はPingメッセージをエンコードする
// クライアントに死活確認のpingを送信するために使用
func EncodePingMessage(sessionID SessionID) []byte {
	header := Header{
		Version:   1,
		SessionID: sessionID.Bytes(),
		Seq:       0,
		Length:    PayloadHeaderSize,
		Timestamp: uint32(time.Now().UnixMilli() & 0xFFFFFFFF),
	}
	payloadHeader := PayloadHeader{
		DataType: DataTypeControl,
		SubType:  uint8(ControlSubTypePing),
	}

	data := make([]byte, HeaderSize+PayloadHeaderSize)
	copy(data[:HeaderSize], header.Encode())
	copy(data[HeaderSize:], payloadHeader.Encode())
	return data
}

// JoinPayload はルーム参加メッセージのペイロード (16バイト)
//
//	roomID  [16]byte  - ルームID (UUID)
type JoinPayload struct {
	RoomID RoomID
}

var ErrInvalidJoinPayloadSize = errors.New("invalid join payload size")

// ParseJoinPayload はバイト列からJoinPayloadをパースする
func ParseJoinPayload(data []byte) (*JoinPayload, error) {
	if len(data) < JoinPayloadSize {
		return nil, ErrInvalidJoinPayloadSize
	}

	var roomID RoomID
	copy(roomID[:], data[:JoinPayloadSize])

	return &JoinPayload{
		RoomID: roomID,
	}, nil
}

// Encode はJoinPayloadをバイト列にエンコードする
func (j *JoinPayload) Encode() []byte {
	return j.RoomID[:]
}

// サイズ定数
const (
	Position2DSize = 8  // 2 * float32
	PositionSize   = 28 // 7 * 4 bytes (7 float32)
	BoneDataSize   = 17 // 1 (BoneID) + 4 * 4 bytes (4 float32)
)

// Position2D は2D位置データ (8バイト)
//
//	x, y float32 (8) - 位置
type Position2D struct {
	X, Y float32
}

// Position は位置・姿勢データ (28バイト)
//
//	x, y, z      float32 (12) - 位置
//	qx, qy, qz, qw float32 (16) - quaternion
type Position struct {
	X, Y, Z        float32 // 位置
	QX, QY, QZ, QW float32 // quaternion
}

// BoneData は1ボーンのデータ (17バイト)
//
//	boneID         uint8   (1)  - ボーンID
//	qx, qy, qz, qw float32 (16) - quaternion
type BoneData struct {
	BoneID         uint8   // ボーンID
	QX, QY, QZ, QW float32 // quaternion
}

// BoneIDToName はボーンIDからボーン名を取得する
// TODO: 実際のボーン名マッピングを実装
func BoneIDToName(id uint8) string {
	return string(rune('0' + id)) // 仮実装: IDをそのまま文字に変換
}

// BoneNameToID はボーン名からボーンIDを取得する
// TODO: 実際のボーン名マッピングを実装
func BoneNameToID(name string) uint8 {
	if len(name) == 0 {
		return 0
	}
	return uint8(name[0] - '0') // 仮実装: 最初の文字をIDとして返す
}

// Actor2DSpawn は2Dキャラ生成メッセージ
type Actor2DSpawn struct {
	Position Position2D
}

// Actor2DUpdate は2Dキャラ更新メッセージ
type Actor2DUpdate struct {
	Position Position2D
}

// Actor2DDespawn は2Dキャラ削除メッセージ
// 削除対象はヘッダーのsessionIDで特定
type Actor2DDespawn struct{}

// Actor3DSpawn はキャラ生成メッセージ
type Actor3DSpawn struct {
	Position Position
}

// Actor3DUpdate はキャラ更新メッセージ（スーパーユーザー用）
//
//	bitmask  [16]byte - 変更ボーンのビットマスク (128ボーン対応)
//	position Position - 位置・姿勢
//	bones    []BoneData - ボーンデータ（可変長）
type Actor3DUpdate struct {
	Bitmask  [16]byte // 変更ボーンのビットマスク (128ボーン対応)
	Position Position
	Bones    []BoneData
}

// Actor3DDespawn はキャラ削除メッセージ
// 削除対象はヘッダーのsessionIDで特定
type Actor3DDespawn struct{}

// InputPayload はユーザー入力 (4バイト)
//
//	keyMask uint32 (4) - キー入力ビットマスク
type InputPayload struct {
	KeyMask uint32
}

// エラー定義
var (
	ErrInvalidPosition2DData    = errors.New("invalid position2d data: expected 8 bytes")
	ErrInvalidPositionSize      = errors.New("invalid position size")
	ErrInvalidBoneDataSize      = errors.New("invalid bone data size")
	ErrInvalidActor2DSpawnSize  = errors.New("invalid actor2d spawn size")
	ErrInvalidActor2DUpdateSize = errors.New("invalid actor2d update size")
	ErrInvalidActor3DSpawnSize  = errors.New("invalid actor3d spawn size")
	ErrInvalidActor3DUpdateSize = errors.New("invalid actor3d update size")
	ErrInvalidInputPayloadSize  = errors.New("invalid input payload size")
)

// ParsePosition2D はバイト列からPosition2Dをパースする
func ParsePosition2D(data []byte) (*Position2D, error) {
	if len(data) < Position2DSize {
		return nil, ErrInvalidPosition2DData
	}

	return &Position2D{
		X: math.Float32frombits(byteOrder.Uint32(data[0:4])),
		Y: math.Float32frombits(byteOrder.Uint32(data[4:8])),
	}, nil
}

// Encode はPosition2Dをバイト列にエンコードする
func (p *Position2D) Encode() []byte {
	buf := make([]byte, Position2DSize)
	byteOrder.PutUint32(buf[0:4], math.Float32bits(p.X))
	byteOrder.PutUint32(buf[4:8], math.Float32bits(p.Y))
	return buf
}

// ParsePosition はバイト列からPositionをパースする
func ParsePosition(data []byte) (*Position, error) {
	if len(data) < PositionSize {
		return nil, ErrInvalidPositionSize
	}

	return &Position{
		X:  math.Float32frombits(byteOrder.Uint32(data[0:4])),
		Y:  math.Float32frombits(byteOrder.Uint32(data[4:8])),
		Z:  math.Float32frombits(byteOrder.Uint32(data[8:12])),
		QX: math.Float32frombits(byteOrder.Uint32(data[12:16])),
		QY: math.Float32frombits(byteOrder.Uint32(data[16:20])),
		QZ: math.Float32frombits(byteOrder.Uint32(data[20:24])),
		QW: math.Float32frombits(byteOrder.Uint32(data[24:28])),
	}, nil
}

// Encode はPositionをバイト列にエンコードする
func (p *Position) Encode() []byte {
	data := make([]byte, PositionSize)
	byteOrder.PutUint32(data[0:4], math.Float32bits(p.X))
	byteOrder.PutUint32(data[4:8], math.Float32bits(p.Y))
	byteOrder.PutUint32(data[8:12], math.Float32bits(p.Z))
	byteOrder.PutUint32(data[12:16], math.Float32bits(p.QX))
	byteOrder.PutUint32(data[16:20], math.Float32bits(p.QY))
	byteOrder.PutUint32(data[20:24], math.Float32bits(p.QZ))
	byteOrder.PutUint32(data[24:28], math.Float32bits(p.QW))
	return data
}

// ParseBoneData はバイト列からBoneDataをパースする
func ParseBoneData(data []byte) (*BoneData, error) {
	if len(data) < BoneDataSize {
		return nil, ErrInvalidBoneDataSize
	}

	return &BoneData{
		BoneID: data[0],
		QX:     math.Float32frombits(byteOrder.Uint32(data[1:5])),
		QY:     math.Float32frombits(byteOrder.Uint32(data[5:9])),
		QZ:     math.Float32frombits(byteOrder.Uint32(data[9:13])),
		QW:     math.Float32frombits(byteOrder.Uint32(data[13:17])),
	}, nil
}

// Encode はBoneDataをバイト列にエンコードする
func (b *BoneData) Encode() []byte {
	data := make([]byte, BoneDataSize)
	data[0] = b.BoneID
	byteOrder.PutUint32(data[1:5], math.Float32bits(b.QX))
	byteOrder.PutUint32(data[5:9], math.Float32bits(b.QY))
	byteOrder.PutUint32(data[9:13], math.Float32bits(b.QZ))
	byteOrder.PutUint32(data[13:17], math.Float32bits(b.QW))
	return data
}

// ParseActor2DSpawn はバイト列からActor2DSpawnをパースする
func ParseActor2DSpawn(data []byte) (*Actor2DSpawn, error) {
	if len(data) < Position2DSize {
		return nil, ErrInvalidActor2DSpawnSize
	}

	pos, err := ParsePosition2D(data)
	if err != nil {
		return nil, err
	}

	return &Actor2DSpawn{
		Position: *pos,
	}, nil
}

// Encode はActor2DSpawnをバイト列にエンコードする
func (a *Actor2DSpawn) Encode() []byte {
	return a.Position.Encode()
}

// ParseActor2DUpdate はバイト列からActor2DUpdateをパースする
func ParseActor2DUpdate(data []byte) (*Actor2DUpdate, error) {
	if len(data) < Position2DSize {
		return nil, ErrInvalidActor2DUpdateSize
	}

	pos, err := ParsePosition2D(data)
	if err != nil {
		return nil, err
	}

	return &Actor2DUpdate{
		Position: *pos,
	}, nil
}

// Encode はActor2DUpdateをバイト列にエンコードする
func (a *Actor2DUpdate) Encode() []byte {
	return a.Position.Encode()
}

// ParseActor3DSpawn はバイト列からActor3DSpawnをパースする
func ParseActor3DSpawn(data []byte) (*Actor3DSpawn, error) {
	if len(data) < PositionSize {
		return nil, ErrInvalidActor3DSpawnSize
	}

	pos, err := ParsePosition(data)
	if err != nil {
		return nil, err
	}

	return &Actor3DSpawn{
		Position: *pos,
	}, nil
}

// Encode はActor3DSpawnをバイト列にエンコードする
func (a *Actor3DSpawn) Encode() []byte {
	return a.Position.Encode()
}

// BitmaskSize はビットマスクのサイズ（16バイト = 128ボーン対応）
const BitmaskSize = 16

// ParseActor3DUpdate はバイト列からActor3DUpdateをパースする
func ParseActor3DUpdate(data []byte) (*Actor3DUpdate, error) {
	minSize := BitmaskSize + PositionSize
	if len(data) < minSize {
		return nil, ErrInvalidActor3DUpdateSize
	}

	var bitmask [16]byte
	copy(bitmask[:], data[0:BitmaskSize])

	pos, err := ParsePosition(data[BitmaskSize:])
	if err != nil {
		return nil, err
	}

	// ビットマスクから有効なボーン数をカウント
	boneCount := countSetBits(bitmask)
	bones := make([]BoneData, 0, boneCount)

	offset := BitmaskSize + PositionSize
	for i := 0; i < boneCount; i++ {
		if offset+BoneDataSize > len(data) {
			return nil, ErrInvalidActor3DUpdateSize
		}
		bone, err := ParseBoneData(data[offset:])
		if err != nil {
			return nil, err
		}
		bones = append(bones, *bone)
		offset += BoneDataSize
	}

	return &Actor3DUpdate{
		Bitmask:  bitmask,
		Position: *pos,
		Bones:    bones,
	}, nil
}

// Encode はActor3DUpdateをバイト列にエンコードする
func (a *Actor3DUpdate) Encode() []byte {
	size := BitmaskSize + PositionSize + len(a.Bones)*BoneDataSize
	data := make([]byte, size)

	copy(data[0:BitmaskSize], a.Bitmask[:])
	copy(data[BitmaskSize:BitmaskSize+PositionSize], a.Position.Encode())

	offset := BitmaskSize + PositionSize
	for _, bone := range a.Bones {
		copy(data[offset:offset+BoneDataSize], bone.Encode())
		offset += BoneDataSize
	}

	return data
}

// countSetBits はビットマスク内の1のビット数をカウントする
func countSetBits(bitmask [16]byte) int {
	count := 0
	for _, b := range bitmask {
		for b != 0 {
			count += int(b & 1)
			b >>= 1
		}
	}
	return count
}

// InputPayloadSize はInputPayloadのサイズ
const InputPayloadSize = 4

// ParseInputPayload はバイト列からInputPayloadをパースする
func ParseInputPayload(data []byte) (*InputPayload, error) {
	if len(data) < InputPayloadSize {
		return nil, ErrInvalidInputPayloadSize
	}

	return &InputPayload{
		KeyMask: byteOrder.Uint32(data[0:4]),
	}, nil
}

// Encode はInputPayloadをバイト列にエンコードする
func (i *InputPayload) Encode() []byte {
	data := make([]byte, InputPayloadSize)
	byteOrder.PutUint32(data[0:4], i.KeyMask)
	return data
}
