package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/touka-aoi/paralle-vs-single/server"
	"github.com/touka-aoi/paralle-vs-single/server/domain"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	slog.SetLogLoggerLevel(slog.LevelDebug)

	addr := "localhost:9091"

	// 初期化
	pubsub := domain.NewSimplePubSub()
	defaultRoomID := domain.RoomID("default")
	roomManager := domain.NewSimpleRoomManager(defaultRoomID)
	echoApp := domain.NewEchoApplication()
	room := domain.NewRoom(defaultRoomID, pubsub, echoApp)

	handler := server.Route(pubsub, roomManager)
	srv := server.NewServer(addr, handler)

	// 起動
	go func() {
		if err := room.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "room error", "err", err)
		}
	}()
	go func() {
		if err := srv.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.ErrorContext(ctx, "server error", "err", err)
		}
	}()
	slog.Info("server started", "addr", addr)
	time.Sleep(100 * time.Millisecond)

	// ループバックテスト
	conn, _, err := websocket.Dial(ctx, "ws://"+addr+"/ws", nil)
	if err != nil {
		slog.Error("failed to connect", "err", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	testMessage := []byte("Hello, WebSocket!")
	if err := conn.Write(ctx, websocket.MessageText, testMessage); err != nil {
		slog.Error("failed to write", "err", err)
		return
	}
	slog.Info("sent message", "data", string(testMessage))

	_, received, err := conn.Read(ctx)
	if err != nil {
		slog.Error("failed to read", "err", err)
		return
	}
	slog.Info("received message", "data", string(received))

	if string(received) == string(testMessage) {
		slog.Info("LOOPBACK TEST PASSED!")
	} else {
		slog.Error("LOOPBACK TEST FAILED", "expected", string(testMessage), "actual", string(received))
	}

	// クリーンアップ
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}
	slog.Info("test complete")
}
