package application

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"

	"flourish/server/domain"
	"flourish/server/domain/crdt"

	"github.com/google/uuid"
)

// RGAStateStore はRGAスナップショットの永続化を担う。
type RGAStateStore interface {
	SaveRGA(ctx context.Context, entryID uuid.UUID, snap crdt.RGASnapshot) error
	LoadRGA(ctx context.Context, entryID uuid.UUID) (crdt.RGASnapshot, error)
	ListRGAEntryIDs(ctx context.Context) ([]uuid.UUID, error)
}

// EntryProjector はイベントからRGAを適用し、Entryのビューを更新する。
type EntryProjector struct {
	rgas          map[uuid.UUID]*crdt.RGA
	mu            sync.Mutex
	entryStore    domain.EntryStore
	rgaStateStore RGAStateStore
	markdownDir   string
	log           *slog.Logger
}

func NewEntryProjector(entryStore domain.EntryStore, rgaStateStore RGAStateStore, markdownDir string, log *slog.Logger) *EntryProjector {
	os.MkdirAll(markdownDir, 0o755)
	return &EntryProjector{
		rgas:          make(map[uuid.UUID]*crdt.RGA),
		entryStore:    entryStore,
		rgaStateStore: rgaStateStore,
		markdownDir:   markdownDir,
		log:           log,
	}
}

// Apply はopをRGAに適用し、Entryを更新する。適用が拒否された場合はfalseを返す。
func (p *EntryProjector) Apply(ctx context.Context, entryID uuid.UUID, payload []byte) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	op, err := crdt.OperationFromPayload(payload)
	if err != nil {
		p.log.Error("projector: payload変換失敗", "entryID", entryID, "error", err)
		return false
	}

	rga, ok := p.rgas[entryID]
	if !ok {
		// サーバー側RGAはゼロUUIDでよい（Tickは使わない）
		rga = crdt.NewRGA(uuid.Nil)
		p.rgas[entryID] = rga
	}

	// 非認証deleteによる認証ノード削除はスキップ（opはイベントストアに記録済み）
	if op.OpType == crdt.OpDelete && !op.Authenticated && rga.IsNodeAuthenticated(op.NodeID) {
		p.log.Warn("projector: 非認証deleteを無視", "entryID", entryID, "nodeID", op.NodeID)
		return false
	}

	rga.Apply(op)

	text := rga.Text()
	title, content := deriveFields(text)

	entry, err := p.entryStore.FindByID(ctx, entryID)
	if err != nil {
		p.log.Error("projector: entry取得失敗", "entryID", entryID, "error", err)
		return false
	}
	entry.Title = title
	entry.Content = content
	entry.Text = text

	if err := p.entryStore.Save(ctx, entry); err != nil {
		p.log.Error("projector: entry保存失敗", "entryID", entryID, "error", err)
		return false
	}

	if err := p.rgaStateStore.SaveRGA(ctx, entryID, rga.Export()); err != nil {
		p.log.Error("projector: RGA状態保存失敗", "entryID", entryID, "error", err)
	}

	p.saveMarkdown(entryID, text)
	return true
}

// IsNodeAuthenticated は指定エントリのノードが認証済みかどうかを返す。
func (p *EntryProjector) IsNodeAuthenticated(entryID uuid.UUID, nodeID crdt.NodeID) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	rga, ok := p.rgas[entryID]
	if !ok {
		return true // RGA未ロード = 安全側
	}
	return rga.IsNodeAuthenticated(nodeID)
}

// Restore はEventStoreの全opからRGAを再構築し、Entryを更新する。
// entryIDsにはEventStoreに存在する全エントリIDを渡す。
func (p *EntryProjector) Restore(ctx context.Context, eventStore domain.EventStore, entryIDs []uuid.UUID) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// RGAスナップショットから復元
	snapIDs, err := p.rgaStateStore.ListRGAEntryIDs(ctx)
	if err != nil {
		return err
	}

	for _, entryID := range snapIDs {
		snap, err := p.rgaStateStore.LoadRGA(ctx, entryID)
		if err != nil {
			p.log.Warn("projector: RGAスナップショット読み込み失敗、opログから復元します", "entryID", entryID, "error", err)
			continue
		}
		rga, err := crdt.ImportRGA(snap)
		if err != nil {
			p.log.Warn("projector: RGAインポート失敗", "entryID", entryID, "error", err)
			continue
		}
		p.rgas[entryID] = rga
	}

	// EventStoreの全エントリのopを再生して差分適用
	for _, entryID := range entryIDs {
		events, err := eventStore.ListAfter(ctx, entryID, 0)
		if err != nil {
			return err
		}
		rga, ok := p.rgas[entryID]
		if !ok {
			rga = crdt.NewRGA(uuid.Nil)
			p.rgas[entryID] = rga
		}
		for _, ev := range events {
			if ev.EventType != domain.EventCRDTOp {
				continue
			}
			op, err := crdt.OperationFromPayload(ev.Payload)
			if err != nil {
				p.log.Warn("projector: op変換失敗", "entryID", entryID, "error", err)
				continue
			}
			rga.Apply(op) // 冪等なので重複適用しても問題ない
		}

		text := rga.Text()
		title, content := deriveFields(text)

		entry, err := p.entryStore.FindByID(ctx, entryID)
		if err != nil {
			p.log.Warn("projector: entry取得失敗、スキップ", "entryID", entryID, "error", err)
			continue
		}
		entry.Title = title
		entry.Content = content
		entry.Text = text
		if err := p.entryStore.Save(ctx, entry); err != nil {
			return err
		}
		if err := p.rgaStateStore.SaveRGA(ctx, entryID, rga.Export()); err != nil {
			p.log.Warn("projector: RGA状態保存失敗", "entryID", entryID, "error", err)
		}

		p.saveMarkdown(entryID, text)
	}

	return nil
}

func (p *EntryProjector) saveMarkdown(entryID uuid.UUID, text string) {
	path := filepath.Join(p.markdownDir, entryID.String()+".md")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		p.log.Error(fmt.Sprintf("projector: markdown保存失敗: %s", err), "entryID", entryID)
	}
}

// deriveFields はテキストからTitle/Contentを導出する。
func deriveFields(text string) (title, content string) {
	lines := strings.SplitN(text, "\n", 2)
	title = lines[0]
	if utf8.RuneCountInString(text) > 200 {
		runes := []rune(text)
		content = string(runes[:200])
	} else {
		content = text
	}
	return
}
