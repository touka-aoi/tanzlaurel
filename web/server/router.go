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
	authHandler *handler.Auth,
) http.Handler {
	mux := http.NewServeMux()

	health := handler.NewHealth()
	entry := handler.NewEntry(entryStore)
	ws := handler.NewWS(syncService, projector, authHandler, log)

	// CSRF保護（state-changing APIに適用）
	csrf := http.NewCrossOriginProtection()

	mux.Handle("GET /api/health", health)
	mux.HandleFunc("GET /api/entries", entry.List)
	mux.Handle("POST /api/entries", csrf.Handler(http.HandlerFunc(entry.Create)))
	if authHandler != nil {
		deleteHandler := csrf.Handler(authHandler.CFAccessMiddleware(handler.RequireAuth(http.HandlerFunc(entry.Delete))))
		mux.Handle("POST /api/admin/entries/{id}/delete", deleteHandler)
		mux.HandleFunc("POST /api/admin/login", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusFound)
		})
	}
	mux.HandleFunc("GET /api/entries/{id}", entry.Get)
	mux.Handle("GET /api/ws", ws)

	// 認証エンドポイント
	if authHandler != nil {
		mux.Handle("POST /api/ws-ticket", csrf.Handler(authHandler.CFAccessMiddleware(http.HandlerFunc(authHandler.WSTicket))))
		mux.HandleFunc("GET /api/auth/status", authHandler.CFAccessMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if handler.IsAuthenticated(r.Context()) {
				w.Write([]byte(`{"authenticated":true}`))
			} else {
				w.Write([]byte(`{"authenticated":false}`))
			}
		})).ServeHTTP)
	}

	return logger.HTTPMiddleware(log)(mux)
}
