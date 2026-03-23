package auth

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// TicketStore はWSチケットのin-memory管理を行う。
type TicketStore struct {
	mu      sync.Mutex
	tickets map[string]time.Time
	ttl     time.Duration
}

// NewTicketStore は新しいTicketStoreを生成する。
func NewTicketStore(ttl time.Duration) *TicketStore {
	return &TicketStore{
		tickets: make(map[string]time.Time),
		ttl:     ttl,
	}
}

// Issue は新しいチケットを発行する。
func (s *TicketStore) Issue() string {
	b := make([]byte, 32)
	rand.Read(b)
	ticket := hex.EncodeToString(b)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanup()
	s.tickets[ticket] = time.Now().Add(s.ttl)
	return ticket
}

// Redeem はチケットを消費する。有効なら true、無効/期限切れなら false。
func (s *TicketStore) Redeem(ticket string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cleanup()
	exp, ok := s.tickets[ticket]
	if !ok {
		return false
	}
	delete(s.tickets, ticket)
	return time.Now().Before(exp)
}

// cleanup は期限切れチケットを削除する。ロック保持前提。
func (s *TicketStore) cleanup() {
	now := time.Now()
	for k, exp := range s.tickets {
		if now.After(exp) {
			delete(s.tickets, k)
		}
	}
}
