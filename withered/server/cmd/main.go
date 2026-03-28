package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"withered/server"
	"withered/server/application"
	"withered/server/domain"
	"withered/utils"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := utils.GetEnvDefault("ADDR", "localhost")
	port := utils.GetEnvDefault("PORT", "9090")

	// PubSub初期化
	pubsub := domain.NewSimplePubSub()

	// デフォルトルーム設定（固定のUUID: 00000000-0000-0000-0000-000000000001）
	defaultRoomID := domain.RoomID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	roomManager := domain.NewSimpleRoomManager(defaultRoomID)

	// Roomを作成して起動
	app := application.NewWitheredApplication()
	room := domain.NewRoom(defaultRoomID, pubsub, app)
	go func() {
		if err := room.Run(ctx); err != nil {
			slog.ErrorContext(ctx, "room error", "err", err)
		}
	}()

	handler := server.Route(pubsub, roomManager)
	s := server.NewServer(fmt.Sprintf("%s:%s", addr, port), handler)

	go func() {
		if err := s.Serve(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http single error: %v", err)
		}
	}()
	slog.InfoContext(ctx, "server listening", "addr", addr+":"+port)

	<-ctx.Done()
	slog.InfoContext(ctx, "shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "graceful shutdown failed", "error", err)
		if err := s.Close(); err != nil {
			slog.ErrorContext(ctx, "forced close failed", "error", err)
		}
	}
	slog.InfoContext(ctx, "server shutdown complete")
}
