package server

import (
	"log/slog"
	"net/http"

	"flourish/server/application"
	"flourish/server/domain"
	"flourish/server/handler"
	"flourish/server/logger"
)

// NewRouter はHTTPルーターを構築する。
func NewRouter(
	log *slog.Logger,
	entryStore domain.EntryStore,
	syncService *application.SyncService,
	projector *application.EntryProjector,
) http.Handler {
	mux := http.NewServeMux()

	health := handler.NewHealth()
	entry := handler.NewEntry(entryStore)
	ws := handler.NewWS(syncService, projector, log)

	mux.Handle("GET /api/health", health)
	mux.HandleFunc("GET /api/entries", entry.List)
	mux.HandleFunc("POST /api/entries", entry.Create)
	mux.HandleFunc("DELETE /api/entries/{id}", entry.Delete)
	mux.HandleFunc("GET /api/entries/{id}", entry.Get)
	mux.Handle("GET /api/ws", ws)

	return logger.HTTPMiddleware(log)(mux)
}
