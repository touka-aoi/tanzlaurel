package memory

import (
	"errors"
	"fmt"
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

func NewStore() *Store {
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

func (s *Store) applyBuff(cmd *state.Buff, ts time.Time) (*domain.BuffResult, error) {
	room, err := s.getRoom(cmd.RoomID)
	if err != nil {
		return nil, err
	}
	stats, err := s.statsForRoom(room)
	if err != nil {
		return nil, err
	}
	targetIDs := cmd.TargetIDs
	if len(targetIDs) == 0 {
		targetIDs = room.MemberIDs
	}

	affected := make([]domain.PlayerSnapshot, 0, len(targetIDs))
	for _, id := range targetIDs {
		player, err := s.getPlayer(id)
		if err != nil {
			return nil, err
		}
		buff := domain.ActiveBuff{
			Buff:      cmd.Buff,
			ExpiresAt: ts.Add(cmd.Buff.Duration),
		}
		player.ActiveBuffs = append(player.ActiveBuffs, buff)
		player.LastUpdated = ts
		affected = append(affected, copyPlayer(player))
	}

	room.LastUpdated = ts
	stats.ActiveBuffHistogram[cmd.Buff.BuffID] += len(affected)
	stats.InteractionCount += len(affected)
	stats.LastUpdated = ts

	return &domain.BuffResult{
		AffectedPlayers: affected,
		Room:            copyRoom(room),
		Stats:           copyRoomStats(stats),
	}, nil
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

func (s *Store) applyTrade(cmd *state.Trade, ts time.Time) (*domain.TradeResult, error) {
	initiator, err := s.getPlayer(cmd.UserID)
	if err != nil {
		return nil, err
	}
	partner, err := s.getPlayer(cmd.PartnerID)
	if err != nil {
		return nil, err
	}
	room, err := s.getRoom(cmd.RoomID)
	if err != nil {
		return nil, err
	}

	if err := applyInventoryChanges(initiator, cmd.Offer, -1); err != nil {
		return nil, err
	}
	if err := applyInventoryChanges(partner, cmd.Request, -1); err != nil {
		return nil, err
	}
	if err := applyInventoryChanges(initiator, cmd.Request, +1); err != nil {
		return nil, err
	}
	if err := applyInventoryChanges(partner, cmd.Offer, +1); err != nil {
		return nil, err
	}

	initiator.LastUpdated = ts
	partner.LastUpdated = ts

	room.LastUpdated = ts
	stats, err := s.statsForRoom(room)
	if err != nil {
		return nil, err
	}
	stats.InteractionCount++
	stats.LastUpdated = ts

	return &domain.TradeResult{
		Initiator: copyPlayer(initiator),
		Partner:   copyPlayer(partner),
		Ledger: domain.TradeLedger{
			Confirmed: true,
			Initiator: copyPlayer(initiator),
			Partner:   copyPlayer(partner),
		},
		Stats: copyRoomStats(stats),
	}, nil
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
	stats := &domain.RoomStats{
		RoomID:              room.RoomID,
		ActiveBuffHistogram: make(map[string]int),
	}
	stats.TotalHealth = s.recomputeTotalHealth(room)
	stats.LastUpdated = room.LastUpdated
	for _, id := range room.MemberIDs {
		if player, ok := s.players[id]; ok {
			for _, buff := range player.ActiveBuffs {
				stats.ActiveBuffHistogram[buff.Buff.BuffID]++
			}
		}
	}
	return stats
}

func copyRoomStats(src *domain.RoomStats) domain.RoomStats {
	if src == nil {
		return domain.RoomStats{
			ActiveBuffHistogram: make(map[string]int),
		}
	}
	dst := *src
	if src.ActiveBuffHistogram == nil {
		dst.ActiveBuffHistogram = make(map[string]int)
	} else {
		dst.ActiveBuffHistogram = make(map[string]int, len(src.ActiveBuffHistogram))
		for k, v := range src.ActiveBuffHistogram {
			dst.ActiveBuffHistogram[k] = v
		}
	}
	return dst
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

func applyInventoryChanges(player *domain.PlayerSnapshot, changes []domain.ItemChange, sign int) error {
	if len(changes) == 0 {
		return nil
	}
	current := make(map[string]int, len(player.Inventory))
	order := make([]string, 0, len(player.Inventory))
	for _, entry := range player.Inventory {
		current[entry.ItemID] = entry.Quantity
		order = append(order, entry.ItemID)
	}

	newIDs := make(map[string]struct{})
	for _, change := range changes {
		if change.QuantityDelta <= 0 {
			return fmt.Errorf("invalid quantity delta for item=%s", change.ItemID)
		}
		qty, ok := current[change.ItemID]
		if !ok {
			if sign < 0 {
				return fmt.Errorf("inventory missing: player=%s item=%s", player.PlayerID, change.ItemID)
			}
			current[change.ItemID] = change.QuantityDelta
			if _, exists := newIDs[change.ItemID]; !exists {
				order = append(order, change.ItemID)
				newIDs[change.ItemID] = struct{}{}
			}
			continue
		}
		next := qty + sign*change.QuantityDelta
		if next < 0 {
			return fmt.Errorf("inventory underflow: player=%s item=%s", player.PlayerID, change.ItemID)
		}
		current[change.ItemID] = next
	}

	updated := make([]domain.InventoryEntry, 0, len(order))
	for _, id := range order {
		qty := current[id]
		if qty <= 0 {
			continue
		}
		updated = append(updated, domain.InventoryEntry{
			ItemID:   id,
			Quantity: qty,
		})
	}
	player.Inventory = updated
	return nil
}
