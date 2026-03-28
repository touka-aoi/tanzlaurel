package domain

import (
	"crypto/rand"
	"encoding/base64"
	"sync/atomic"
	"time"
)

type SessionID string

// NewSessionID は暗号学的に安全なSessionIDを生成する
func NewSessionID() SessionID {
	var b [16]byte
	rand.Read(b[:])
	return SessionID(base64.RawURLEncoding.EncodeToString(b[:]))
}

// Bytes はSessionIDを16バイトのバイト列に変換する
// 前提: SessionIDは常にNewSessionID()で生成される内部型であり、
// 外部からの入力で直接構築されない。そのためデコードエラーは発生しない。
// 詳細はADR-007を参照。
func (id SessionID) Bytes() [16]byte {
	var b [16]byte
	decoded, _ := base64.RawURLEncoding.DecodeString(string(id))
	copy(b[:], decoded)
	return b
}

// SessionIDFromBytes はバイト列からSessionIDを生成する
func SessionIDFromBytes(b [16]byte) SessionID {
	return SessionID(base64.RawURLEncoding.EncodeToString(b[:]))
}

func (id SessionID) String() string { return string(id) }

// Session は1接続の論理的な接続状態を表す構造体です。
type Session struct {
	id SessionID

	// activity
	lastRead  atomic.Int64
	lastWrite atomic.Int64
	lastPong  atomic.Int64

	// backpressure ( 未実装 )
	//sendQ *BoundedQueue[[]byte] // bounded ring buffer

	// lifecycle
	closed atomic.Bool
}

func NewSession() *Session {
	s := &Session{
		id: NewSessionID(),
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

func (s *Session) Close() bool {
	if s.closed.CompareAndSwap(false, true) {
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

func (s *Session) ID() SessionID {
	return s.id
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
