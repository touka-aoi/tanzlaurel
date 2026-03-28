package domain

import (
	"time"

	"github.com/google/uuid"
)

// Entry はブログエントリを表す。
type Entry struct {
	ID        uuid.UUID
	Title     string
	Content   string
	Thumbnail *string
	Text      string
	CreatedAt time.Time
	UpdatedAt time.Time
	Deleted   bool
}

// EntryListItem は一覧表示用のエントリ。textフィールドを除外する。
type EntryListItem struct {
	ID        uuid.UUID
	Title     string
	Content   string
	Thumbnail *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewEntry は空のエントリを新規作成する。
func NewEntry() Entry {
	now := time.Now().UTC()
	return Entry{
		ID:        uuid.New(),
		Title:     "",
		Content:   "",
		Text:      "",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ToListItem はEntryListItemに変換する。
func (e Entry) ToListItem() EntryListItem {
	return EntryListItem{
		ID:        e.ID,
		Title:     e.Title,
		Content:   e.Content,
		Thumbnail: e.Thumbnail,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}
