package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

// ProblemDetail はRFC 9457準拠のエラーレスポンス。
type ProblemDetail struct {
	Type     string `json:"type"`
	Status   int    `json:"status"`
	Title    string `json:"title"`
	Instance string `json:"instance"`
}

func writeProblem(w http.ResponseWriter, status int, errType, title string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ProblemDetail{
		Type:     errType,
		Status:   status,
		Title:    title,
		Instance: "urn:uuid:" + uuid.New().String(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
