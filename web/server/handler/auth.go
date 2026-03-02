package handler

import (
	"context"
	"encoding/json"
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
	adminUser     string
	adminPassword string
	jwt           *auth.JWTService
	tickets       *auth.TicketStore
}

// NewAuth は新しいAuthハンドラーを生成する。
func NewAuth(adminUser, adminPassword string, jwt *auth.JWTService, tickets *auth.TicketStore) *Auth {
	return &Auth{
		adminUser:     adminUser,
		adminPassword: adminPassword,
		jwt:           jwt,
		tickets:       tickets,
	}
}

// Login はBasic認証でJWT Cookieを発行する。
func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != a.adminUser || pass != a.adminPassword {
		w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token, err := a.jwt.Sign(user)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   3600,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// Logout はCookieを削除する。
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   0,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// WSTicket はJWT Cookie検証後にWSチケットを発行する。
func (a *Auth) WSTicket(w http.ResponseWriter, r *http.Request) {
	if !IsAuthenticated(r.Context()) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	ticket := a.tickets.Issue()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ticket": ticket})
}

// JWTMiddleware はJWT Cookieを検証してコンテキストに認証状態を設定するミドルウェア。
func (a *Auth) JWTMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated := false

		cookie, err := r.Cookie("token")
		if err == nil && cookie.Value != "" {
			if _, err := a.jwt.Verify(cookie.Value); err == nil {
				authenticated = true
			}
		}

		ctx := context.WithValue(r.Context(), authContextKey, authenticated)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth はJWTMiddlewareの後に使い、未認証なら403を返すミドルウェア。
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
