package main

import (
	"os"

	"flourish/server"
	"flourish/server/adapter/memory"
	"flourish/server/logger"
)

func main() {
	logLevel := envOrDefault("LOG_LEVEL", "info")
	addr := envOrDefault("ADDRESS", ":8080")

	cfg := logger.Config{
		Level:       logLevel,
		ServiceName: "crdt-blog",
		Version:     "v0.1.0",
		Environment: envOrDefault("ENV", "development"),
	}

	log := logger.New(cfg)
	logger.PrintBanner(cfg, addr, "")

	entryStore := memory.NewEntryStore()
	router := server.NewRouter(log, entryStore)
	srv := server.New(addr, router, log)

	if err := srv.Run(); err != nil {
		log.Error("server error", "error", err)
		os.Exit(1)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
