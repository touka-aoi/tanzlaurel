package application

import (
	"context"
	"encoding/binary"
	"math"
	"slices"

	"withered/server/application/protocol"
)

type Codec[W any] interface {
	EncodeSnapshot(ctx context.Context, world W) ([]byte, error)
	EncodeDelta(ctx context.Context, world W) ([]byte, error)
}

// TODO: Codecを満たしているかどうかのガードを設定する。
type WitheredCodec struct{}

func (c WitheredCodec) EncodeSnapshot(ctx context.Context, world *ShootingWorld) ([]byte, error) {
	body := encodeWitheredSnapshot(world)
	header := &protocol.PayloadHeader{
		DataType: protocol.DataTypeSnapshot,
		SubType:  0,
	}
	out := make([]byte, 0, protocol.PayloadHeaderSize+len(body))
	out = append(out, header.Encode()...)
	out = append(out, body...)
	return out, nil
}

func (c WitheredCodec) EncodeDelta(ctx context.Context, world *ShootingWorld) ([]byte, error) {
	panic("not implemented")
}

func encodeWitheredSnapshot(world *ShootingWorld) []byte {
	ids := make([]EntityID, 0, len(world.Entities))
	for id := range world.Entities {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	buf := make([]byte, 0, 256)
	buf = binary.LittleEndian.AppendUint16(buf, uint16(len(ids)))
	for _, id := range ids {
		buf = encodeEntity(world, id, buf)
	}
	return buf
}

const (
	//TODO: PositionMask に命名を変更したい
	MPos  = 1 << 0
	MVel  = 1 << 1
	MHP   = 1 << 2
	MLife = 1 << 3
	MTTL  = 1 << 4
	MOwn  = 1 << 5
	MTag  = 1 << 6
)

func encodeEntity(w *ShootingWorld, id EntityID, buf []byte) []byte {
	mask := uint8(0)
	if _, ok := w.Position[id]; ok {
		mask |= MPos
	}
	if _, ok := w.Velocity[id]; ok {
		mask |= MVel
	}
	if _, ok := w.Health[id]; ok {
		mask |= MHP
	}
	if _, ok := w.LifeState[id]; ok {
		mask |= MLife
	}
	if _, ok := w.TTL[id]; ok {
		mask |= MTTL
	}
	if _, ok := w.Owner[id]; ok {
		mask |= MOwn
	}
	if _, ok := w.Tag[id]; ok {
		mask |= MTag
	}

	buf = binary.LittleEndian.AppendUint16(buf, uint16(id))
	buf = append(buf, mask)

	if mask&MPos != 0 {
		p := w.Position[id]
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(p.X))
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(p.Y))
	}
	if mask&MVel != 0 {
		v := w.Velocity[id]
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(v.X))
		buf = binary.LittleEndian.AppendUint32(buf, math.Float32bits(v.Y))
	}
	if mask&MHP != 0 {
		buf = append(buf, w.Health[id].HP)
	}
	if mask&MLife != 0 {
		buf = append(buf, uint8(w.LifeState[id].State))
	}
	if mask&MTTL != 0 {
		buf = binary.LittleEndian.AppendUint16(buf, w.TTL[id])
	}
	if mask&MOwn != 0 {
		buf = binary.LittleEndian.AppendUint16(buf, uint16(w.Owner[id]))
	}
	if mask&MTag != 0 {
		buf = append(buf, uint8(w.Tag[id]))
	}

	return buf
}

func decodeEntity(buf []byte) {
	panic("not implemented")
}
