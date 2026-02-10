package domain

import (
	"math"
	"testing"
)

func TestHeaderRoundTrip(t *testing.T) {
	original := &Header{
		Version:   1,
		SessionID: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		Seq:       100,
		Length:    256,
		Timestamp: 1234567890,
	}

	encoded := original.Encode()
	if len(encoded) != HeaderSize {
		t.Errorf("encoded size = %d, want %d", len(encoded), HeaderSize)
	}

	decoded, err := ParseHeader(encoded)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("Version = %d, want %d", decoded.Version, original.Version)
	}
	if decoded.SessionID != original.SessionID {
		t.Errorf("SessionID = %d, want %d", decoded.SessionID, original.SessionID)
	}
	if decoded.Seq != original.Seq {
		t.Errorf("Seq = %d, want %d", decoded.Seq, original.Seq)
	}
	if decoded.Length != original.Length {
		t.Errorf("Length = %d, want %d", decoded.Length, original.Length)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp = %d, want %d", decoded.Timestamp, original.Timestamp)
	}
}

func TestPayloadHeaderRoundTrip(t *testing.T) {
	original := &PayloadHeader{
		DataType: DataTypeActor2D,
		SubType:  uint8(ActorSubTypeSpawn),
	}

	encoded := original.Encode()
	if len(encoded) != PayloadHeaderSize {
		t.Errorf("encoded size = %d, want %d", len(encoded), PayloadHeaderSize)
	}

	decoded, err := ParsePayloadHeader(encoded)
	if err != nil {
		t.Fatalf("ParsePayloadHeader failed: %v", err)
	}

	if decoded.DataType != original.DataType {
		t.Errorf("DataType = %d, want %d", decoded.DataType, original.DataType)
	}
	if decoded.SubType != original.SubType {
		t.Errorf("SubType = %d, want %d", decoded.SubType, original.SubType)
	}
}

func TestPosition2DRoundTrip(t *testing.T) {
	original := &Position2D{X: 1.5, Y: -2.5}

	encoded := original.Encode()
	if len(encoded) != Position2DSize {
		t.Fatalf("encoded size = %d, want %d", len(encoded), Position2DSize)
	}

	decoded, err := ParsePosition2D(encoded)
	if err != nil {
		t.Fatalf("ParsePosition2D failed: %v", err)
	}

	if decoded.X != original.X || decoded.Y != original.Y {
		t.Errorf("decoded = %+v, want %+v", decoded, original)
	}
}

func TestPosition2DParseInvalidData(t *testing.T) {
	_, err := ParsePosition2D([]byte{0x01, 0x02, 0x03})
	if err != ErrInvalidPosition2DData {
		t.Errorf("expected ErrInvalidPosition2DData, got %v", err)
	}
}

func TestPosition2DZero(t *testing.T) {
	pos := &Position2D{X: 0, Y: 0}
	encoded := pos.Encode()
	decoded, err := ParsePosition2D(encoded)
	if err != nil {
		t.Fatalf("ParsePosition2D failed: %v", err)
	}

	if decoded.X != 0 || decoded.Y != 0 {
		t.Errorf("decoded = %+v, want {0, 0}", decoded)
	}
}

func TestPositionRoundTrip(t *testing.T) {
	original := &Position{
		X: 1.5, Y: 2.5, Z: 3.5,
		QX: 0.1, QY: 0.2, QZ: 0.3, QW: 0.9,
	}

	encoded := original.Encode()
	if len(encoded) != PositionSize {
		t.Errorf("encoded size = %d, want %d", len(encoded), PositionSize)
	}

	decoded, err := ParsePosition(encoded)
	if err != nil {
		t.Fatalf("ParsePosition failed: %v", err)
	}

	if !floatEqual(decoded.X, original.X) {
		t.Errorf("X = %f, want %f", decoded.X, original.X)
	}
	if !floatEqual(decoded.Y, original.Y) {
		t.Errorf("Y = %f, want %f", decoded.Y, original.Y)
	}
	if !floatEqual(decoded.Z, original.Z) {
		t.Errorf("Z = %f, want %f", decoded.Z, original.Z)
	}
	if !floatEqual(decoded.QX, original.QX) {
		t.Errorf("QX = %f, want %f", decoded.QX, original.QX)
	}
	if !floatEqual(decoded.QY, original.QY) {
		t.Errorf("QY = %f, want %f", decoded.QY, original.QY)
	}
	if !floatEqual(decoded.QZ, original.QZ) {
		t.Errorf("QZ = %f, want %f", decoded.QZ, original.QZ)
	}
	if !floatEqual(decoded.QW, original.QW) {
		t.Errorf("QW = %f, want %f", decoded.QW, original.QW)
	}
}

func TestBoneDataRoundTrip(t *testing.T) {
	original := &BoneData{
		BoneID: 5,
		QX:     0.1, QY: 0.2, QZ: 0.3, QW: 0.9,
	}

	encoded := original.Encode()
	if len(encoded) != BoneDataSize {
		t.Errorf("encoded size = %d, want %d", len(encoded), BoneDataSize)
	}

	decoded, err := ParseBoneData(encoded)
	if err != nil {
		t.Fatalf("ParseBoneData failed: %v", err)
	}

	if decoded.BoneID != original.BoneID {
		t.Errorf("BoneID = %d, want %d", decoded.BoneID, original.BoneID)
	}
	if !floatEqual(decoded.QX, original.QX) {
		t.Errorf("QX = %f, want %f", decoded.QX, original.QX)
	}
	if !floatEqual(decoded.QY, original.QY) {
		t.Errorf("QY = %f, want %f", decoded.QY, original.QY)
	}
	if !floatEqual(decoded.QZ, original.QZ) {
		t.Errorf("QZ = %f, want %f", decoded.QZ, original.QZ)
	}
	if !floatEqual(decoded.QW, original.QW) {
		t.Errorf("QW = %f, want %f", decoded.QW, original.QW)
	}
}

func TestActor3DSpawnRoundTrip(t *testing.T) {
	original := &Actor3DSpawn{
		Position: Position{
			X: 10.0, Y: 20.0, Z: 30.0,
			QX: 0.0, QY: 0.0, QZ: 0.0, QW: 1.0,
		},
	}

	encoded := original.Encode()
	if len(encoded) != PositionSize {
		t.Errorf("encoded size = %d, want %d", len(encoded), PositionSize)
	}

	decoded, err := ParseActor3DSpawn(encoded)
	if err != nil {
		t.Fatalf("ParseActor3DSpawn failed: %v", err)
	}

	if !floatEqual(decoded.Position.X, original.Position.X) {
		t.Errorf("Position.X = %f, want %f", decoded.Position.X, original.Position.X)
	}
	if !floatEqual(decoded.Position.QW, original.Position.QW) {
		t.Errorf("Position.QW = %f, want %f", decoded.Position.QW, original.Position.QW)
	}
}

func TestActor3DUpdateRoundTrip(t *testing.T) {
	original := &Actor3DUpdate{
		Bitmask: [16]byte{0x03, 0x00}, // ボーン0と1が有効
		Position: Position{
			X: 5.0, Y: 10.0, Z: 15.0,
			QX: 0.0, QY: 0.0, QZ: 0.0, QW: 1.0,
		},
		Bones: []BoneData{
			{BoneID: 0, QX: 0.1, QY: 0.2, QZ: 0.3, QW: 0.9},
			{BoneID: 1, QX: 0.4, QY: 0.5, QZ: 0.6, QW: 0.7},
		},
	}

	encoded := original.Encode()
	expectedSize := BitmaskSize + PositionSize + 2*BoneDataSize
	if len(encoded) != expectedSize {
		t.Errorf("encoded size = %d, want %d", len(encoded), expectedSize)
	}

	decoded, err := ParseActor3DUpdate(encoded)
	if err != nil {
		t.Fatalf("ParseActor3DUpdate failed: %v", err)
	}

	if decoded.Bitmask != original.Bitmask {
		t.Errorf("Bitmask = %v, want %v", decoded.Bitmask, original.Bitmask)
	}
	if !floatEqual(decoded.Position.X, original.Position.X) {
		t.Errorf("Position.X = %f, want %f", decoded.Position.X, original.Position.X)
	}
	if len(decoded.Bones) != len(original.Bones) {
		t.Fatalf("Bones length = %d, want %d", len(decoded.Bones), len(original.Bones))
	}
	for i, bone := range decoded.Bones {
		if bone.BoneID != original.Bones[i].BoneID {
			t.Errorf("Bones[%d].BoneID = %d, want %d", i, bone.BoneID, original.Bones[i].BoneID)
		}
		if !floatEqual(bone.QX, original.Bones[i].QX) {
			t.Errorf("Bones[%d].QX = %f, want %f", i, bone.QX, original.Bones[i].QX)
		}
	}
}

func TestInputPayloadRoundTrip(t *testing.T) {
	original := &InputPayload{
		KeyMask: 0x12345678,
	}

	encoded := original.Encode()
	if len(encoded) != InputPayloadSize {
		t.Errorf("encoded size = %d, want %d", len(encoded), InputPayloadSize)
	}

	decoded, err := ParseInputPayload(encoded)
	if err != nil {
		t.Fatalf("ParseInputPayload failed: %v", err)
	}

	if decoded.KeyMask != original.KeyMask {
		t.Errorf("KeyMask = %d, want %d", decoded.KeyMask, original.KeyMask)
	}
}

func TestCountSetBits(t *testing.T) {
	tests := []struct {
		name     string
		bitmask  [16]byte
		expected int
	}{
		{"empty", [16]byte{}, 0},
		{"one bit", [16]byte{0x01}, 1},
		{"two bits", [16]byte{0x03}, 2},
		{"full byte", [16]byte{0xFF}, 8},
		{"multiple bytes", [16]byte{0xFF, 0x01}, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countSetBits(tt.bitmask)
			if result != tt.expected {
				t.Errorf("countSetBits(%v) = %d, want %d", tt.bitmask, result, tt.expected)
			}
		})
	}
}

func TestParseHeaderInvalidSize(t *testing.T) {
	data := make([]byte, HeaderSize-1)
	_, err := ParseHeader(data)
	if err != ErrInvalidHeaderSize {
		t.Errorf("expected ErrInvalidHeaderSize, got %v", err)
	}
}

func TestParsePayloadHeaderInvalidSize(t *testing.T) {
	data := make([]byte, PayloadHeaderSize-1)
	_, err := ParsePayloadHeader(data)
	if err != ErrInvalidPayloadSize {
		t.Errorf("expected ErrInvalidPayloadSize, got %v", err)
	}
}

func TestParsePositionInvalidSize(t *testing.T) {
	data := make([]byte, PositionSize-1)
	_, err := ParsePosition(data)
	if err != ErrInvalidPositionSize {
		t.Errorf("expected ErrInvalidPositionSize, got %v", err)
	}
}

func TestParseBoneDataInvalidSize(t *testing.T) {
	data := make([]byte, BoneDataSize-1)
	_, err := ParseBoneData(data)
	if err != ErrInvalidBoneDataSize {
		t.Errorf("expected ErrInvalidBoneDataSize, got %v", err)
	}
}

// floatEqual compares two float32 values with tolerance
func floatEqual(a, b float32) bool {
	const epsilon = 1e-6
	return math.Abs(float64(a-b)) < epsilon
}
