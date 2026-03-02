package main

import (
	"context"
	"os"
	"path/filepath"

	"flourish/server"
	"flourish/server/adapter/jsonfile"
	"flourish/server/application"
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

	dataDir := envOrDefault("DATA_DIR", "data")

	entryStore, err := jsonfile.NewEntryStore(dataDir)
	if err != nil {
		log.Error("entry store初期化エラー", "error", err)
		os.Exit(1)
	}
	eventStore, err := jsonfile.NewEventStore(dataDir)
	if err != nil {
		log.Error("event store初期化エラー", "error", err)
		os.Exit(1)
	}
	rgaStateStore, err := jsonfile.NewRGAStateStore(dataDir)
	if err != nil {
		log.Error("rga state store初期化エラー", "error", err)
		os.Exit(1)
	}

	syncService := application.NewSyncService(eventStore)
	markdownDir := filepath.Join(dataDir, "markdown")
	projector := application.NewEntryProjector(entryStore, rgaStateStore, markdownDir, log)

	// 起動時にEventStoreからRGA復元
	entryIDs := eventStore.EntryIDs()
	if err := projector.Restore(context.Background(), eventStore, entryIDs); err != nil {
		log.Error("projector復元エラー", "error", err)
		os.Exit(1)
	}

	router := server.NewRouter(log, entryStore, syncService, projector)
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
