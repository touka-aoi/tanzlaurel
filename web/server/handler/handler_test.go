package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"flourish/server/adapter/memory"
	"flourish/server/domain"
	"flourish/server/handler"
)

func TestHealth(t *testing.T) {
	h := handler.NewHealth()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ステータスコードが200であるべき: got %d", rec.Code)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("statusが'ok'であるべき: got %q", body["status"])
	}
}

func TestEntryHandler_Create(t *testing.T) {
	store := memory.NewEntryStore()
	h := handler.NewEntry(store)

	req := httptest.NewRequest(http.MethodPost, "/api/entries", nil)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("ステータスコードが201であるべき: got %d", rec.Code)
	}

	var body handler.EntryCreatedResponse
	json.NewDecoder(rec.Body).Decode(&body)
	if body.ID == "" {
		t.Error("IDが空であるべきではない")
	}
	if body.Title != "" {
		t.Errorf("タイトルが空文字列であるべき: got %q", body.Title)
	}
}

func TestEntryHandler_List(t *testing.T) {
	store := memory.NewEntryStore()
	entry := domain.NewEntry()
	entry.Title = "テスト"
	store.Save(context.Background(), entry)

	h := handler.NewEntry(store)
	req := httptest.NewRequest(http.MethodGet, "/api/entries", nil)
	rec := httptest.NewRecorder()

	h.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("ステータスコードが200であるべき: got %d", rec.Code)
	}

	var body handler.EntryListResponse
	json.NewDecoder(rec.Body).Decode(&body)
	if len(body.Entries) != 1 {
		t.Errorf("エントリが1件であるべき: got %d", len(body.Entries))
	}
}

func TestEntryHandler_Delete(t *testing.T) {
	store := memory.NewEntryStore()
	entry := domain.NewEntry()
	store.Save(context.Background(), entry)

	h := handler.NewEntry(store)
	req := httptest.NewRequest(http.MethodDelete, "/api/entries/"+entry.ID.String(), nil)
	req.SetPathValue("id", entry.ID.String())
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("ステータスコードが204であるべき: got %d", rec.Code)
	}
}

func TestEntryHandler_Delete_NotFound(t *testing.T) {
	store := memory.NewEntryStore()
	h := handler.NewEntry(store)

	req := httptest.NewRequest(http.MethodDelete, "/api/entries/00000000-0000-0000-0000-000000000001", nil)
	req.SetPathValue("id", "00000000-0000-0000-0000-000000000001")
	rec := httptest.NewRecorder()

	h.Delete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("ステータスコードが404であるべき: got %d", rec.Code)
	}

	var body handler.ProblemDetail
	json.NewDecoder(rec.Body).Decode(&body)
	if body.Type != "error:entry_not_found" {
		t.Errorf("error_typeが'error:entry_not_found'であるべき: got %q", body.Type)
	}
}
