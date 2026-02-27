package protocol

import "errors"

// --- MsgType ---

type MsgType uint64

const (
	MsgTypeRoomMessage MsgType = 0x00
	MsgTypePing        MsgType = 0x01
	MsgTypePong        MsgType = 0x02
	MsgTypeAssign      MsgType = 0x03
)

// --- RoomMsgType ---

type RoomMsgType uint64

const (
	RoomMsgTypeJoin    RoomMsgType = 0x00
	RoomMsgTypeLeave   RoomMsgType = 0x01
	RoomMsgTypeAppData RoomMsgType = 0x02
)

// --- エラー ---

var (
	ErrTooShort      = errors.New("buffer too short")
	ErrInvalidRoomID = errors.New("invalid room ID length")
)

// --- パーサー ---

// ParseTransportHeader はTransportHeader（MsgType + TotalLen）を読み取る。
// 戻り値: msgType, totalLen（ペイロード長）, n（消費バイト数）
func ParseTransportHeader(buf []byte) (msgType MsgType, totalLen uint64, n int, err error) {
	mt, n1, err := ReadVarint(buf)
	if err != nil {
		return 0, 0, 0, err
	}

	tl, n2, err := ReadVarint(buf[n1:])
	if err != nil {
		return 0, 0, 0, err
	}

	return MsgType(mt), tl, n1 + n2, nil
}

// ParseRoomID はLength-Prefixed Stringとして RoomID を読み取る。
// 戻り値: roomID, n（消費バイト数）
func ParseRoomID(buf []byte) (roomID string, n int, err error) {
	length, n1, err := ReadVarint(buf)
	if err != nil {
		return "", 0, err
	}

	end := n1 + int(length)
	if len(buf) < end {
		return "", 0, ErrInvalidRoomID
	}

	return string(buf[n1:end]), end, nil
}

// ParseRoomHeader は RoomMsgType を読み取る。
// 戻り値: roomMsgType, n（消費バイト数）
func ParseRoomHeader(buf []byte) (roomMsgType RoomMsgType, n int, err error) {
	v, n1, err := ReadVarint(buf)
	if err != nil {
		return 0, 0, err
	}
	return RoomMsgType(v), n1, nil
}

// --- エンコーダー ---

// EncodePing はPingメッセージをエンコードする。
// [MsgType=0x01][TotalLen=0]
func EncodePing() []byte {
	buf := make([]byte, 0, 2)
	buf = AppendVarint(buf, uint64(MsgTypePing))
	buf = AppendVarint(buf, 0) // payload なし
	return buf
}

// EncodePong はPongメッセージをエンコードする。
// [MsgType=0x02][TotalLen=0]
func EncodePong() []byte {
	buf := make([]byte, 0, 2)
	buf = AppendVarint(buf, uint64(MsgTypePong))
	buf = AppendVarint(buf, 0) // payload なし
	return buf
}

// EncodeAssign はAssignメッセージ（SessionID通知）をエンコードする。
// [MsgType=0x03][TotalLen=16][SessionID: 16bytes]
func EncodeAssign(sessionID [16]byte) []byte {
	buf := make([]byte, 0, 2+16)
	buf = AppendVarint(buf, uint64(MsgTypeAssign))
	buf = AppendVarint(buf, 16) // SessionID = 16バイト
	buf = append(buf, sessionID[:]...)
	return buf
}

// EncodeRoomMessage はRoomMessageをエンコードする。
// [MsgType=0x00][TotalLen][RoomIDLen][RoomID][RoomMsgType][AppPayload]
func EncodeRoomMessage(roomID string, roomMsgType RoomMsgType, appPayload []byte) []byte {
	// ペイロード部分を先に構築してTotalLenを算出
	payload := make([]byte, 0, VarintLen(uint64(len(roomID)))+len(roomID)+VarintLen(uint64(roomMsgType))+len(appPayload))
	payload = AppendVarint(payload, uint64(len(roomID)))
	payload = append(payload, roomID...)
	payload = AppendVarint(payload, uint64(roomMsgType))
	payload = append(payload, appPayload...)

	buf := make([]byte, 0, VarintLen(uint64(MsgTypeRoomMessage))+VarintLen(uint64(len(payload)))+len(payload))
	buf = AppendVarint(buf, uint64(MsgTypeRoomMessage))
	buf = AppendVarint(buf, uint64(len(payload)))
	buf = append(buf, payload...)
	return buf
}
