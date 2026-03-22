package memory_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"flourish/server/adapter/memory"
	"flourish/server/domain"
)

func TestEntryStore_SaveAndFindByID(t *testing.T) {
	store := memory.NewEntryStore()
	ctx := context.Background()

	entry := domain.NewEntry()
	entry.Title = "テスト記事"

	if err := store.Save(ctx, entry); err != nil {
		t.Fatal(err)
	}

	got, err := store.FindByID(ctx, entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "テスト記事" {
		t.Errorf("タイトルが不一致: got %q", got.Title)
	}
}

func TestEntryStore_FindByID_NotFound(t *testing.T) {
	store := memory.NewEntryStore()
	ctx := context.Background()

	_, err := store.FindByID(ctx, uuid.New())
	if !errors.Is(err, domain.ErrEntryNotFound) {
		t.Errorf("ErrEntryNotFoundを返すべき: got %v", err)
	}
}

func TestEntryStore_FindByID_Deleted(t *testing.T) {
	store := memory.NewEntryStore()
	ctx := context.Background()

	entry := domain.NewEntry()
	store.Save(ctx, entry)
	store.Delete(ctx, entry.ID)

	_, err := store.FindByID(ctx, entry.ID)
	if !errors.Is(err, domain.ErrEntryDeleted) {
		t.Errorf("ErrEntryDeletedを返すべき: got %v", err)
	}
}

func TestEntryStore_List_ExcludesDeleted(t *testing.T) {
	store := memory.NewEntryStore()
	ctx := context.Background()

	e1 := domain.NewEntry()
	e1.Title = "記事1"
	e2 := domain.NewEntry()
	e2.Title = "記事2"
	store.Save(ctx, e1)
	store.Save(ctx, e2)
	store.Delete(ctx, e1.ID)

	items, err := store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("削除済みを除いた一覧は1件: got %d", len(items))
	}
	if items[0].Title != "記事2" {
		t.Errorf("記事2が返されるべき: got %q", items[0].Title)
	}
}
