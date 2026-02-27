package protocol

import "testing"

func TestVarintRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		bytes int // 期待されるエンコードバイト数
	}{
		{"zero", 0, 1},
		{"max_1byte", 63, 1},
		{"min_2byte", 64, 2},
		{"max_2byte", 16383, 2},
		{"min_4byte", 16384, 4},
		{"max_4byte", 1073741823, 4},
		{"min_8byte", 1073741824, 8},
		{"large_8byte", 1<<62 - 1, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// VarintLen
			if got := VarintLen(tt.value); got != tt.bytes {
				t.Errorf("VarintLen(%d) = %d, want %d", tt.value, got, tt.bytes)
			}

			// Encode
			buf := AppendVarint(nil, tt.value)
			if len(buf) != tt.bytes {
				t.Fatalf("AppendVarint: len = %d, want %d", len(buf), tt.bytes)
			}

			// Decode
			value, n, err := ReadVarint(buf)
			if err != nil {
				t.Fatalf("ReadVarint: %v", err)
			}
			if n != tt.bytes {
				t.Errorf("ReadVarint: n = %d, want %d", n, tt.bytes)
			}
			if value != tt.value {
				t.Errorf("ReadVarint: value = %d, want %d", value, tt.value)
			}
		})
	}
}

func TestReadVarintEmptyBuffer(t *testing.T) {
	_, _, err := ReadVarint(nil)
	if err != ErrVarintTooShort {
		t.Errorf("expected ErrVarintTooShort, got %v", err)
	}
}

func TestReadVarintTruncatedBuffer(t *testing.T) {
	// 2バイトvarintの先頭1バイトだけ渡す
	buf := AppendVarint(nil, 100) // 100 > 63 → 2バイト
	_, _, err := ReadVarint(buf[:1])
	if err != ErrVarintTooShort {
		t.Errorf("expected ErrVarintTooShort, got %v", err)
	}
}

func TestAppendVarintToExistingBuffer(t *testing.T) {
	buf := []byte{0xFF}
	buf = AppendVarint(buf, 42)
	if buf[0] != 0xFF {
		t.Errorf("existing byte overwritten: got %x", buf[0])
	}
	value, _, err := ReadVarint(buf[1:])
	if err != nil {
		t.Fatalf("ReadVarint: %v", err)
	}
	if value != 42 {
		t.Errorf("value = %d, want 42", value)
	}
}
