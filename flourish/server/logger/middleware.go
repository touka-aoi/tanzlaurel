package logger

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
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

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// HTTPMiddleware はリクエスト/レスポンスのログを出力するミドルウェア。
func HTTPMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			reqID := uuid.New().String()

			// トレースIDをログに付与
			attrs := []any{
				"http.request.method", r.Method,
				"url.path", r.URL.Path,
				"client.address", r.RemoteAddr,
				"user_agent.original", r.UserAgent(),
				"http.request.body.size", r.ContentLength,
				"requestId", reqID,
			}
			if spanCtx := trace.SpanFromContext(r.Context()).SpanContext(); spanCtx.HasTraceID() {
				attrs = append(attrs, "trace_id", spanCtx.TraceID().String())
				attrs = append(attrs, "span_id", spanCtx.SpanID().String())
			}

			log.InfoContext(r.Context(), "request received", attrs...)

			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)

			respAttrs := []any{
				"http.request.method", r.Method,
				"url.path", r.URL.Path,
				"http.response.status_code", rw.statusCode,
				"http.server.request.duration", time.Since(start).Seconds(),
				"requestId", reqID,
			}
			if spanCtx := trace.SpanFromContext(r.Context()).SpanContext(); spanCtx.HasTraceID() {
				respAttrs = append(respAttrs, "trace_id", spanCtx.TraceID().String())
				respAttrs = append(respAttrs, "span_id", spanCtx.SpanID().String())
			}

			log.InfoContext(r.Context(), "response sent", respAttrs...)
		})
	}
}
