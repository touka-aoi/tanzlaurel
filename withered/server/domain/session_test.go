package domain

import "testing"

// TestNewSession_InitializesTimestamps は NewSession がタイムスタンプを初期化することを確認します。
func TestNewSession_InitializesTimestamps(t *testing.T) {
	s := NewSession()

	if s.lastRead.Load() == 0 {
		t.Errorf("lastRead is not initialized")
	}
	if s.lastWrite.Load() == 0 {
		t.Errorf("lastWrite is not initialized")
	}
	if s.lastPong.Load() == 0 {
		t.Errorf("lastPong is not initialized")
	}
}
