package jsonfile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"flourish/server/domain/crdt"

	"github.com/google/uuid"
)

// RGAStateStore はRGAスナップショットのJSONファイル永続化。
// data/rga_states/{entryID}.json に保存する。
type RGAStateStore struct {
	dir string
}

func NewRGAStateStore(dataDir string) (*RGAStateStore, error) {
	dir := filepath.Join(dataDir, "rga_states")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create rga_states dir: %w", err)
	}
	return &RGAStateStore{dir: dir}, nil
}

func (s *RGAStateStore) SaveRGA(_ context.Context, entryID uuid.UUID, snap crdt.RGASnapshot) error {
	path := filepath.Join(s.dir, entryID.String()+".json")
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *RGAStateStore) LoadRGA(_ context.Context, entryID uuid.UUID) (crdt.RGASnapshot, error) {
	path := filepath.Join(s.dir, entryID.String()+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return crdt.RGASnapshot{}, err
	}
	var snap crdt.RGASnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return crdt.RGASnapshot{}, err
	}
	return snap, nil
}

func (s *RGAStateStore) ListRGAEntryIDs(_ context.Context) ([]uuid.UUID, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []uuid.UUID
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		idStr := strings.TrimSuffix(e.Name(), ".json")
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}
