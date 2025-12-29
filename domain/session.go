package domain

import (
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// Session は1接続の論理的な接続状態を表す構造体です。
type Session struct {
	ID string

	// activity
	lastRead  atomic.Int64
	lastWrite atomic.Int64
	lastPong  atomic.Int64

	// backpressure ( 未実装 )
	//sendQ *BoundedQueue[[]byte] // bounded ring buffer

	// lifecycle
	closed      atomic.Bool
	closeReason atomic.Uint32
}

func NewSession() *Session {
	s := &Session{
		ID: uuid.NewString(),
	}
	now := time.Now().UnixNano()
	s.lastRead.Store(now)
	s.lastWrite.Store(now)
	s.lastPong.Store(now)
	return s
}

func (s *Session) TouchRead() {
	s.lastRead.Store(time.Now().UnixNano())
}

func (s *Session) TouchWrite() {
	s.lastWrite.Store(time.Now().UnixNano())
}

func (s *Session) TouchPong() {
	s.lastPong.Store(time.Now().UnixNano())
}

func (s *Session) Close(reason uint32) bool {
	if s.closed.CompareAndSwap(false, true) {
		s.closeReason.Store(reason)
		return true
	}
	return false
}

func (s *Session) IsIdle(timeout time.Duration) (bool, IdleReason) {
	if timeout <= 0 {
		return false, IdleDisabled
	}
	var reason IdleReason
	if s.IsReadIdle(timeout) {
		reason |= IdleRead
	}
	if s.IsWriteIdle(timeout) {
		reason |= IdleWrite
	}
	if s.IsPongIdle(timeout) {
		reason |= IdlePong
	}
	return reason != IdleNone, reason
}

func (s *Session) IsReadIdle(timeout time.Duration) bool {
	return isIdleSince(unixNanoToTime(s.lastRead.Load()), timeout)
}

func (s *Session) IsWriteIdle(timeout time.Duration) bool {
	return isIdleSince(unixNanoToTime(s.lastWrite.Load()), timeout)
}

func (s *Session) IsPongIdle(timeout time.Duration) bool {
	return isIdleSince(unixNanoToTime(s.lastPong.Load()), timeout)
}

func (s *Session) IsClosed() bool {
	return s.closed.Load()
}

func isIdleSince(last time.Time, timeout time.Duration) bool {
	return time.Since(last) > timeout
}

func unixNanoToTime(nano int64) time.Time {
	return time.Unix(0, nano)
}
