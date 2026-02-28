package logger

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// HTTPMiddleware はリクエスト/レスポンスのログを出力するミドルウェア。
func HTTPMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := uuid.New().String()

			log.Info("request received",
				"http.request.method", r.Method,
				"url.path", r.URL.Path,
				"client.address", r.RemoteAddr,
				"user_agent.original", r.UserAgent(),
				"http.request.body.size", r.ContentLength,
				"requestId", reqID,
			)

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			log.Info("response sent",
				"http.request.method", r.Method,
				"url.path", r.URL.Path,
				"http.response.status_code", rw.statusCode,
				"http.server.request.duration", time.Since(start).Seconds(),
				"requestId", reqID,
			)
		})
	}
}
