package domain

import (
	"encoding/binary"
	"errors"
	"math"
)

const Position2DSize = 8 // 2 * float32

type Position2D struct {
	X, Y float32
}

var ErrInvalidPosition2DData = errors.New("invalid position2d data: expected 8 bytes")

func ParsePosition2D(data []byte) (*Position2D, error) {
	if len(data) < Position2DSize {
		return nil, ErrInvalidPosition2DData
	}

	return &Position2D{
		X: math.Float32frombits(binary.LittleEndian.Uint32(data[0:4])),
		Y: math.Float32frombits(binary.LittleEndian.Uint32(data[4:8])),
	}, nil
}

func (p *Position2D) Encode() []byte {
	buf := make([]byte, Position2DSize)
	binary.LittleEndian.PutUint32(buf[0:4], math.Float32bits(p.X))
	binary.LittleEndian.PutUint32(buf[4:8], math.Float32bits(p.Y))
	return buf
}
