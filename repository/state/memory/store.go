package memory

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/touka-aoi/paralle-vs-single/domain"
	"github.com/touka-aoi/paralle-vs-single/repository/state"
)

var (
	ErrPlayerNotFound = errors.New("memory: player not found")
	ErrRoomNotFound   = errors.New("memory: room not found")
)

type Store struct {
	players   map[string]*domain.PlayerSnapshot
	rooms     map[string]*domain.RoomSnapshot
	roomStats map[string]*domain.RoomStats
}

func newStore() *Store {
	store := &Store{
		players:   make(map[string]*domain.PlayerSnapshot),
		rooms:     make(map[string]*domain.RoomSnapshot),
		roomStats: make(map[string]*domain.RoomStats),
	}
	return store
}

func (s *Store) applyMove(cmd *state.Move, ts time.Time) (*domain.MoveResult, error) {
	player, err := s.getPlayer(cmd.UserID)
	if err != nil {
		return nil, err
	}
	player.Position = cmd.NextPosition
	player.LastUpdated = ts
	return &domain.MoveResult{Player: copyPlayer(player)}, nil
}

func (s *Store) applyAttack(cmd *state.Attack, ts time.Time) (*domain.AttackResult, error) {
	attacker, err := s.getPlayer(cmd.UserID)
	if err != nil {
		return nil, err
	}
	target, err := s.getPlayer(cmd.TargetID)
	if err != nil {
		return nil, err
	}
	room, err := s.getRoom(cmd.RoomID)
	if err != nil {
		return nil, err
	}
	stats, err := s.statsForRoom(room)
	if err != nil {
		return nil, err
	}

	target.Health -= cmd.Damage
	if target.Health < 0 {
		target.Health = 0
	}

	attacker.LastUpdated = ts
	target.LastUpdated = ts

	room.LastUpdated = ts
	stats.InteractionCount++
	stats.TotalHealth = s.recomputeTotalHealth(room)
	stats.LastUpdated = ts

	return &domain.AttackResult{
		Attacker: copyPlayer(attacker),
		Target:   copyPlayer(target),
		Room:     copyRoom(room),
		Stats:    copyRoomStats(stats),
	}, nil
}

func (s *Store) registerPlayer(playerID string, roomID string) error {
	player, ok := s.players[playerID]
	if !ok {
		player = &domain.PlayerSnapshot{PlayerID: playerID, Health: 100}
		s.players[playerID] = player
	}
	player.RoomID = roomID

	room, ok := s.rooms[roomID]
	if !ok {
		room = &domain.RoomSnapshot{RoomID: roomID}
		s.rooms[roomID] = &domain.RoomSnapshot{RoomID: roomID}
	}
	if !slices.Contains(room.MemberIDs, playerID) {
		room.MemberIDs = append(room.MemberIDs, playerID)
	}
	s.roomStats[roomID] = s.computeRoomStats(room)
	return nil
}

func (s *Store) getPlayer(id string) (*domain.PlayerSnapshot, error) {
	player, ok := s.players[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrPlayerNotFound, id)
	}
	return player, nil
}

func (s *Store) getRoom(id string) (*domain.RoomSnapshot, error) {
	room, ok := s.rooms[id]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrRoomNotFound, id)
	}
	return room, nil
}

func (s *Store) recomputeTotalHealth(room *domain.RoomSnapshot) int {
	total := 0
	for _, id := range room.MemberIDs {
		if player, ok := s.players[id]; ok {
			total += player.Health
		}
	}
	return total
}

func copyPlayer(src *domain.PlayerSnapshot) domain.PlayerSnapshot {
	if src == nil {
		return domain.PlayerSnapshot{}
	}
	cp := *src
	cp.ActiveBuffs = append([]domain.ActiveBuff(nil), src.ActiveBuffs...)
	cp.Inventory = append([]domain.InventoryEntry(nil), src.Inventory...)
	return cp
}

func copyRoom(src *domain.RoomSnapshot) domain.RoomSnapshot {
	if src == nil {
		return domain.RoomSnapshot{}
	}
	cp := *src
	cp.MemberIDs = append([]string(nil), src.MemberIDs...)
	return cp
}

func (s *Store) computeRoomStats(room *domain.RoomSnapshot) *domain.RoomStats {
	stats := &domain.RoomStats{RoomID: room.RoomID}
	stats.TotalHealth = s.recomputeTotalHealth(room)
	stats.LastUpdated = room.LastUpdated
	return stats
}

func copyRoomStats(src *domain.RoomStats) domain.RoomStats {
	if src == nil {
		return domain.RoomStats{}
	}
	return *src
}

func (s *Store) statsForRoom(room *domain.RoomSnapshot) (*domain.RoomStats, error) {
	if room == nil {
		return nil, fmt.Errorf("memory: room is nil")
	}
	stats, ok := s.roomStats[room.RoomID]
	if !ok {
		return nil, fmt.Errorf("memory: stats not registered for room %s", room.RoomID)
	}
	return stats, nil
}
