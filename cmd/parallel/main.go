package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/service"
	parallelhandler "github.com/touka-aoi/paralle-vs-single/handler/parallel"
	"github.com/touka-aoi/paralle-vs-single/server/parallel"
	"github.com/touka-aoi/paralle-vs-single/utils"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var addr string
	addr = utils.GetEnvDefault("ADDR", "localhost")
	port := utils.GetEnvDefault("PORT", "9090")

	svc, err := service.NewInteractionService()
	if err != nil {
		log.Fatalf("failed to construct service: %v", err)
	}

	h := parallelhandler.NewHandler(svc)
	s := parallel.NewServer(fmt.Sprintf("%s:%s", addr, port), h)
	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http single error: %v", err)
		}
	}()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		if err := s.Close(); err != nil {
			log.Printf("forced close failed: %v", err)
		}
	}
}
