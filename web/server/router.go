package server

import (
	"log/slog"
	"net/http"

	"flourish/server/handler"
	"flourish/server/logger"

	"flourish/server/domain"
)

// NewRouter はHTTPルーターを構築する。
func NewRouter(log *slog.Logger, entryStore domain.EntryStore) http.Handler {
	mux := http.NewServeMux()

	health := handler.NewHealth()
	entry := handler.NewEntry(entryStore)

	mux.Handle("GET /api/health", health)
	mux.HandleFunc("GET /api/entries", entry.List)
	mux.HandleFunc("POST /api/entries", entry.Create)
	mux.HandleFunc("DELETE /api/entries/{id}", entry.Delete)

	return logger.HTTPMiddleware(log)(mux)
}
