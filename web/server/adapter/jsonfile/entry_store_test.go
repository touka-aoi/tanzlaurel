package jsonfile_test

import (
	"testing"

	"github.com/google/uuid"

	"flourish/server/adapter/jsonfile"
	"flourish/server/domain"
)

func TestEntryStore_SaveAndReload(t *testing.T) {
	dir := t.TempDir()

	entryID := uuid.New()

	// 1回目: 保存
	{
		store, err := jsonfile.NewEntryStore(dir)
		if err != nil {
			t.Fatal(err)
		}
		entry := domain.NewEntry()
		entry.ID = entryID
		entry.Title = "Test Title"
		entry.Content = "Test Content"
		entry.Text = "Test Title\nTest Content"
		if err := store.Save(t.Context(), entry); err != nil {
			t.Fatal(err)
		}
	}

	// 2回目: 再読み込み
	{
		store, err := jsonfile.NewEntryStore(dir)
		if err != nil {
			t.Fatal(err)
		}

		entry, err := store.FindByID(t.Context(), entryID)
		if err != nil {
			t.Fatal(err)
		}
		if entry.Title != "Test Title" {
			t.Errorf("Title: got %q, want %q", entry.Title, "Test Title")
		}
		if entry.Text != "Test Title\nTest Content" {
			t.Errorf("Text: got %q, want %q", entry.Text, "Test Title\nTest Content")
		}

		items, err := store.List(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 {
			t.Errorf("list count: got %d, want 1", len(items))
		}
	}
}

func TestEntryStore_Delete(t *testing.T) {
	dir := t.TempDir()
	store, err := jsonfile.NewEntryStore(dir)
	if err != nil {
		t.Fatal(err)
	}

	entry := domain.NewEntry()
	store.Save(t.Context(), entry)

	if err := store.Delete(t.Context(), entry.ID); err != nil {
		t.Fatal(err)
	}

	_, err = store.FindByID(t.Context(), entry.ID)
	if err != domain.ErrEntryDeleted {
		t.Errorf("expected ErrEntryDeleted, got %v", err)
	}

	items, _ := store.List(t.Context())
	if len(items) != 0 {
		t.Errorf("deleted entry should not appear in list: got %d", len(items))
	}
}
