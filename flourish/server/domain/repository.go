package domain

import (
	"context"

	"github.com/google/uuid"
)

// EventStore はイベントの永続化と取得を担う。
type EventStore interface {
	// Append はイベントを追記し、server_seqを採番して返す。
	// 同一request_idのイベントが既に存在する場合は0を返す（重複検知）。
	Append(ctx context.Context, event Event) (serverSeq int64, err error)

	// ListAfter は指定されたserver_seq以降のイベントを取得する。
	ListAfter(ctx context.Context, entryID uuid.UUID, afterSeq int64) ([]Event, error)

	// MaxServerSeq は指定エントリの最大server_seqを返す。
	MaxServerSeq(ctx context.Context, entryID uuid.UUID) (int64, error)
}

// EntryStore はエントリのCRUD操作を担う。
type EntryStore interface {
	// Save はエントリを保存する（作成・更新兼用）。
	Save(ctx context.Context, entry Entry) error

	// FindByID はIDでエントリを取得する。存在しない場合はErrEntryNotFoundを返す。
	FindByID(ctx context.Context, id uuid.UUID) (Entry, error)

	// List は全エントリの一覧を取得する（削除済みを除く）。
	List(ctx context.Context) ([]EntryListItem, error)

	// Delete はエントリを論理削除する。
	Delete(ctx context.Context, id uuid.UUID) error
}
