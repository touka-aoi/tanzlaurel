package protocol

import (
	"bytes"
	"testing"
)

func TestPingRoundTrip(t *testing.T) {
	data := EncodePing()

	msgType, totalLen, n, err := ParseTransportHeader(data)
	if err != nil {
		t.Fatalf("ParseTransportHeader: %v", err)
	}
	if msgType != MsgTypePing {
		t.Errorf("MsgType = %d, want %d", msgType, MsgTypePing)
	}
	if totalLen != 0 {
		t.Errorf("TotalLen = %d, want 0", totalLen)
	}
	if n != len(data) {
		t.Errorf("consumed = %d, want %d", n, len(data))
	}
}

func TestPongRoundTrip(t *testing.T) {
	data := EncodePong()

	msgType, totalLen, n, err := ParseTransportHeader(data)
	if err != nil {
		t.Fatalf("ParseTransportHeader: %v", err)
	}
	if msgType != MsgTypePong {
		t.Errorf("MsgType = %d, want %d", msgType, MsgTypePong)
	}
	if totalLen != 0 {
		t.Errorf("TotalLen = %d, want 0", totalLen)
	}
	if n != len(data) {
		t.Errorf("consumed = %d, want %d", n, len(data))
	}
}

func TestAssignRoundTrip(t *testing.T) {
	sessionID := [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	data := EncodeAssign(sessionID)

	msgType, totalLen, n, err := ParseTransportHeader(data)
	if err != nil {
		t.Fatalf("ParseTransportHeader: %v", err)
	}
	if msgType != MsgTypeAssign {
		t.Errorf("MsgType = %d, want %d", msgType, MsgTypeAssign)
	}
	if totalLen != 16 {
		t.Errorf("TotalLen = %d, want 16", totalLen)
	}

	payload := data[n : n+int(totalLen)]
	if !bytes.Equal(payload, sessionID[:]) {
		t.Errorf("SessionID = %v, want %v", payload, sessionID[:])
	}
}

func TestRoomMessageRoundTrip(t *testing.T) {
	roomID := "test-room"
	appPayload := []byte{0xDE, 0xAD, 0xBE, 0xEF}

	data := EncodeRoomMessage(roomID, RoomMsgTypeAppData, appPayload)

	// TransportHeader
	msgType, totalLen, n, err := ParseTransportHeader(data)
	if err != nil {
		t.Fatalf("ParseTransportHeader: %v", err)
	}
	if msgType != MsgTypeRoomMessage {
		t.Errorf("MsgType = %d, want %d", msgType, MsgTypeRoomMessage)
	}

	payload := data[n : n+int(totalLen)]

	// RoomID
	gotRoomID, n2, err := ParseRoomID(payload)
	if err != nil {
		t.Fatalf("ParseRoomID: %v", err)
	}
	if gotRoomID != roomID {
		t.Errorf("RoomID = %q, want %q", gotRoomID, roomID)
	}

	// RoomHeader
	roomMsgType, n3, err := ParseRoomHeader(payload[n2:])
	if err != nil {
		t.Fatalf("ParseRoomHeader: %v", err)
	}
	if roomMsgType != RoomMsgTypeAppData {
		t.Errorf("RoomMsgType = %d, want %d", roomMsgType, RoomMsgTypeAppData)
	}

	// AppPayload
	gotPayload := payload[n2+n3:]
	if !bytes.Equal(gotPayload, appPayload) {
		t.Errorf("AppPayload = %v, want %v", gotPayload, appPayload)
	}
}

func TestRoomMessageJoinNoPayload(t *testing.T) {
	roomID := "room-1"
	data := EncodeRoomMessage(roomID, RoomMsgTypeJoin, nil)

	_, totalLen, n, err := ParseTransportHeader(data)
	if err != nil {
		t.Fatalf("ParseTransportHeader: %v", err)
	}

	payload := data[n : n+int(totalLen)]

	gotRoomID, n2, err := ParseRoomID(payload)
	if err != nil {
		t.Fatalf("ParseRoomID: %v", err)
	}
	if gotRoomID != roomID {
		t.Errorf("RoomID = %q, want %q", gotRoomID, roomID)
	}

	roomMsgType, n3, err := ParseRoomHeader(payload[n2:])
	if err != nil {
		t.Fatalf("ParseRoomHeader: %v", err)
	}
	if roomMsgType != RoomMsgTypeJoin {
		t.Errorf("RoomMsgType = %d, want %d", roomMsgType, RoomMsgTypeJoin)
	}

	// Joinはpayloadなし
	remaining := payload[n2+n3:]
	if len(remaining) != 0 {
		t.Errorf("remaining = %d bytes, want 0", len(remaining))
	}
}

func TestParseRoomIDInvalidLength(t *testing.T) {
	// RoomIDLen=100 だがデータが足りない
	buf := AppendVarint(nil, 100)
	_, _, err := ParseRoomID(buf)
	if err != ErrInvalidRoomID {
		t.Errorf("expected ErrInvalidRoomID, got %v", err)
	}
}

func TestParseTransportHeaderTooShort(t *testing.T) {
	_, _, _, err := ParseTransportHeader(nil)
	if err == nil {
		t.Error("expected error for empty buffer")
	}
}

func TestParseRoomHeaderTooShort(t *testing.T) {
	_, _, err := ParseRoomHeader(nil)
	if err == nil {
		t.Error("expected error for empty buffer")
	}
}
