package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/touka-aoi/paralle-vs-single/server"
	"github.com/touka-aoi/paralle-vs-single/server/domain"
	"github.com/touka-aoi/paralle-vs-single/utils"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	addr := utils.GetEnvDefault("ADDR", "localhost")
	port := utils.GetEnvDefault("PORT", "9090")

	dispatcher := domain.NewLoopbackDispatcher()
	handler := server.Route(dispatcher)
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
