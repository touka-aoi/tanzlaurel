package domain_test

import (
	"context"
	"testing"
	"time"

	domain "withered/server/domain"
)

func TestHeartbeatService_SendsPingToWriteCh(t *testing.T) {
	session := domain.NewSession()
	writeCh := make(chan []byte, 16)

	hb := domain.NewHeartbeatService(50*time.Millisecond, session, writeCh)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go hb.Run(ctx)

	// 少なくとも1つのpingが送信されることを確認
	select {
	case msg := <-writeCh:
		if msg == nil {
			t.Fatal("received nil message")
		}
		// 新プロトコル: Ping = [MsgType=0x01: 1byte][TotalLen=0: 1byte] = 2バイト
		expectedSize := 2
		if len(msg) != expectedSize {
			t.Fatalf("unexpected message size: got %d, want %d", len(msg), expectedSize)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for ping message")
	}
}

func TestHeartbeatService_StopsOnContextCancel(t *testing.T) {
	session := domain.NewSession()
	writeCh := make(chan []byte, 16)

	hb := domain.NewHeartbeatService(50*time.Millisecond, session, writeCh)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		hb.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// 正常終了
	case <-time.After(1 * time.Second):
		t.Fatal("HeartbeatService did not stop after context cancel")
	}
}

func TestHeartbeatService_DropsWhenWriteChFull(t *testing.T) {
	session := domain.NewSession()
	// バッファサイズ0でwriteChが常に満杯になるようにする
	writeCh := make(chan []byte)

	hb := domain.NewHeartbeatService(50*time.Millisecond, session, writeCh)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		hb.Run(ctx)
		close(done)
	}()

	// ブロックせずにRunが完了する（dropしてpanicしない）ことを確認
	select {
	case <-done:
		// 正常終了
	case <-time.After(1 * time.Second):
		t.Fatal("HeartbeatService blocked on full writeCh")
	}
}
