package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"flourish/server"
	"flourish/server/adapter/jsonfile"
	"flourish/server/application"
	"flourish/server/auth"
	"flourish/server/handler"
	"flourish/server/logger"
	appotel "flourish/server/otel"
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

	// OTel初期化（OTEL_EXPORTER_OTLP_ENDPOINT が設定されている場合のみ有効）
	var otelProviders *appotel.Providers
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otlpEndpoint != "" {
		providers, err := appotel.Setup(context.Background(), appotel.Config{
			ServiceName:    cfg.ServiceName,
			ServiceVersion: cfg.Version,
			Environment:    cfg.Environment,
			OTLPEndpoint:   otlpEndpoint,
		})
		if err != nil {
			slog.Error("OTel初期化エラー", "error", err)
			os.Exit(1)
		}
		otelProviders = providers
		defer otelProviders.Shutdown(context.Background())
	}

	// ロガー作成（OTelが有効ならブリッジハンドラーを追加）
	var log *slog.Logger
	if otelProviders != nil {
		localHandler := logger.NewHandler(cfg)
		otelHandler := appotel.NewSlogHandler(otelProviders.LoggerProvider)
		log = slog.New(logger.NewFanoutHandler(localHandler, otelHandler)).With(
			"service.name", cfg.ServiceName,
			"service.version", cfg.Version,
			"deployment.environment", cfg.Environment,
		)
	} else {
		log = logger.New(cfg)
	}
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

	// 認証セットアップ（CF_ACCESS_TEAM_DOMAIN + CF_ACCESS_AUDIENCE が設定されている場合のみ有効）
	var authHandler *handler.Auth
	cfTeamDomain := os.Getenv("CF_ACCESS_TEAM_DOMAIN")
	cfAudience := os.Getenv("CF_ACCESS_AUDIENCE")
	if cfTeamDomain != "" && cfAudience != "" {
		cfAccess, err := auth.NewCFAccessVerifier(cfTeamDomain, cfAudience)
		if err != nil {
			log.Error("CF Access初期化エラー", "error", err)
			os.Exit(1)
		}
		ticketStore := auth.NewTicketStore(1 * time.Minute)
		authHandler = handler.NewAuth(cfAccess, ticketStore, cfTeamDomain)
		log.Info("CF Access認証有効", "teamDomain", cfTeamDomain)
	} else {
		log.Info("認証無効（CF_ACCESS_TEAM_DOMAIN/CF_ACCESS_AUDIENCE未設定）")
	}

	router := server.NewRouter(log, entryStore, syncService, projector, authHandler)
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
