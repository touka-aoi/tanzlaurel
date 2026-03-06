package handler

import "net/http"

// Health はヘルスチェックハンドラー。
type Health struct{}

func NewHealth() *Health {
	return &Health{}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
