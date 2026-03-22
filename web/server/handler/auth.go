package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"flourish/server/auth"
)

type contextKey string

const authContextKey contextKey = "authenticated"

// IsAuthenticated はコンテキストから認証状態を取得する。
func IsAuthenticated(ctx context.Context) bool {
	v, _ := ctx.Value(authContextKey).(bool)
	return v
}

// Auth は認証関連のハンドラー。
type Auth struct {
	cfAccess *auth.CFAccessVerifier
	tickets  *auth.TicketStore
}

// NewAuth は新しいAuthハンドラーを生成する。
func NewAuth(cfAccess *auth.CFAccessVerifier, tickets *auth.TicketStore) *Auth {
	return &Auth{
		cfAccess: cfAccess,
		tickets:  tickets,
	}
}

// WSTicket はCF Access検証後にWSチケットを発行する。
func (a *Auth) WSTicket(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r.Context()) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ticket := a.tickets.Issue()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ticket": ticket})
}

// CFAccessMiddleware はCF Access JWTを検証してコンテキストに認証状態を設定するミドルウェア。
func (a *Auth) CFAccessMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated := false

		tokenStr := r.Header.Get("Cf-Access-Jwt-Assertion")
		source := "header"
		if tokenStr == "" {
			if cookie, err := r.Cookie("CF_Authorization"); err == nil {
				tokenStr = cookie.Value
				source = "cookie"
			}
		}

		if tokenStr != "" {
			if _, err := a.cfAccess.Verify(tokenStr); err == nil {
				authenticated = true
			} else {
				slog.Warn("CF Access JWT検証失敗", "source", source, "error", err, "path", r.URL.Path)
			}
		} else {
			slog.Warn("CF Access トークンなし", "path", r.URL.Path)
		}

		ctx := context.WithValue(r.Context(), authContextKey, authenticated)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth はCFAccessMiddlewareの後に使い、未認証なら403を返すミドルウェア。
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r.Context()) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RedeemTicket はチケットを検証・消費する。
func (a *Auth) RedeemTicket(ticket string) bool {
	return a.tickets.Redeem(ticket)
}
