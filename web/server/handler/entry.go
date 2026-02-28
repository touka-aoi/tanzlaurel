package handler

import (
	"errors"
	"net/http"
	"time"

	"flourish/server/domain"

	"github.com/google/uuid"
)

// EntryCreatedResponse はエントリ作成レスポンス。
type EntryCreatedResponse struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// EntryListItemResponse は一覧表示用のエントリレスポンス。
type EntryListItemResponse struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Thumbnail *string `json:"thumbnail"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// EntryListResponse はエントリ一覧レスポンス。
type EntryListResponse struct {
	Entries []EntryListItemResponse `json:"entries"`
}

// Entry はエントリのHTTPハンドラー。
type Entry struct {
	store domain.EntryStore
}

func NewEntry(store domain.EntryStore) *Entry {
	return &Entry{store: store}
}

func (h *Entry) Create(w http.ResponseWriter, r *http.Request) {
	entry := domain.NewEntry()
	if err := h.store.Save(r.Context(), entry); err != nil {
		writeProblem(w, http.StatusInternalServerError, "about:blank", "Internal Server Error")
		return
	}

	writeJSON(w, http.StatusCreated, EntryCreatedResponse{
		ID:        entry.ID.String(),
		Title:     entry.Title,
		CreatedAt: entry.CreatedAt.Format(time.RFC3339),
		UpdatedAt: entry.UpdatedAt.Format(time.RFC3339),
	})
}

func (h *Entry) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "about:blank", "Internal Server Error")
		return
	}

	entries := make([]EntryListItemResponse, len(items))
	for i, item := range items {
		entries[i] = EntryListItemResponse{
			ID:        item.ID.String(),
			Title:     item.Title,
			Content:   item.Content,
			Thumbnail: item.Thumbnail,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
			UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
		}
	}

	writeJSON(w, http.StatusOK, EntryListResponse{Entries: entries})
}

func (h *Entry) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "about:blank", "Bad Request")
		return
	}

	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrEntryNotFound) {
			writeProblem(w, http.StatusNotFound, "error:entry_not_found", "Entry Not Found")
			return
		}
		writeProblem(w, http.StatusInternalServerError, "about:blank", "Internal Server Error")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
