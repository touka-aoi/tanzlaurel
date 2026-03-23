package protocol

import "errors"

// QUIC varint (RFC 9000 Section 16)
//
// 先頭2ビットでバイト長を判定:
//   0b00 → 1バイト (6ビット値, max 63)
//   0b01 → 2バイト (14ビット値, max 16383)
//   0b10 → 4バイト (30ビット値, max 1073741823)
//   0b11 → 8バイト (62ビット値, max 4611686018427387903)

var (
	ErrVarintTooShort = errors.New("buffer too short for varint")
	ErrVarintOverflow = errors.New("varint value exceeds maximum (2^62 - 1)")
)

const maxVarint = uint64(1<<62 - 1)

// ReadVarint はバッファからQUIC varintをデコードする。
// value: デコードされた値, n: 消費バイト数
func ReadVarint(buf []byte) (value uint64, n int, err error) {
	if len(buf) == 0 {
		return 0, 0, ErrVarintTooShort
	}

	prefix := buf[0] >> 6
	length := 1 << prefix // 1, 2, 4, 8

	if len(buf) < length {
		return 0, 0, ErrVarintTooShort
	}

	switch length {
	case 1:
		value = uint64(buf[0] & 0x3f)
	case 2:
		value = uint64(buf[0]&0x3f)<<8 | uint64(buf[1])
	case 4:
		value = uint64(buf[0]&0x3f)<<24 | uint64(buf[1])<<16 | uint64(buf[2])<<8 | uint64(buf[3])
	case 8:
		value = uint64(buf[0]&0x3f)<<56 | uint64(buf[1])<<48 | uint64(buf[2])<<40 | uint64(buf[3])<<32 |
			uint64(buf[4])<<24 | uint64(buf[5])<<16 | uint64(buf[6])<<8 | uint64(buf[7])
	}

	return value, length, nil
}

// AppendVarint はバッファにQUIC varintをエンコードして追加する。
func AppendVarint(buf []byte, value uint64) []byte {
	switch {
	case value <= 63:
		return append(buf, byte(value))
	case value <= 16383:
		return append(buf, byte(0x40|value>>8), byte(value))
	case value <= 1073741823:
		return append(buf, byte(0x80|value>>24), byte(value>>16), byte(value>>8), byte(value))
	default:
		return append(buf,
			byte(0xc0|value>>56), byte(value>>48), byte(value>>40), byte(value>>32),
			byte(value>>24), byte(value>>16), byte(value>>8), byte(value))
	}
}

// VarintLen はvalueをエンコードするのに必要なバイト数を返す。
func VarintLen(value uint64) int {
	switch {
	case value <= 63:
		return 1
	case value <= 16383:
		return 2
	case value <= 1073741823:
		return 4
	default:
		return 8
	}
}
