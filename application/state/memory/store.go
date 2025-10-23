package memory

import (
	"errors"
	"fmt"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
)

var (
	ErrPlayerNotFound = errors.New("memory: player not found")
	ErrRoomNotFound   = errors.New("memory: room not found")
)

// Store はインメモリのプレイヤー・ルーム状態を保持する共通ストレージ。
// 並行実装・単一ループ実装は本ストアをラップして利用し、ロック戦略のみを差し替える。
type Store struct {
	players   map[string]*domain.PlayerSnapshot
	rooms     map[string]*domain.RoomSnapshot
	roomStats map[string]*domain.RoomStats
}

// NewStore は初期状態をコピーしつつストアを生成する。
func NewStore(players []domain.PlayerSnapshot, rooms []domain.RoomSnapshot) *Store {
	store := &Store{
		players:   make(map[string]*domain.PlayerSnapshot, len(players)),
		rooms:     make(map[string]*domain.RoomSnapshot, len(rooms)),
		roomStats: make(map[string]*domain.RoomStats, len(rooms)),
	}
	for i := range players {
		p := players[i]
		store.players[p.PlayerID] = clonePlayerPtr(&p)
	}
	for i := range rooms {
		r := rooms[i]
		store.rooms[r.RoomID] = cloneRoomPtr(&r)
	}
	for _, room := range store.rooms {
		store.roomStats[room.RoomID] = store.computeRoomStats(room)
	}
	return store
}

func (s *Store) applyMove(cmd domain.MoveCommand, ts time.Time) (domain.MoveResult, error) {
	player, err := s.getPlayer(cmd.ActorID)
	if err != nil {
		return domain.MoveResult{}, err
	}
	player.Position = cmd.NextPosition
	player.LastUpdated = ts
	return domain.MoveResult{Player: copyPlayer(player)}, nil
}

// tsを受け取るのはこの関数がstoreすることのみに集中するため
func (s *Store) applyBuff(cmd domain.BuffCommand, ts time.Time) (domain.BuffResult, error) {
	room, err := s.getRoom(cmd.RoomID)
	if err != nil {
		return domain.BuffResult{}, err
	}
	stats, err := s.statsForRoom(room)
	if err != nil {
		return domain.BuffResult{}, err
	}
	targetIDs := cmd.TargetIDs
	if len(targetIDs) == 0 {
		targetIDs = room.MemberIDs
	}

	affected := make([]domain.PlayerSnapshot, 0, len(targetIDs))
	for _, id := range targetIDs {
		player, err := s.getPlayer(id)
		if err != nil {
			return domain.BuffResult{}, err
		}
		buff := domain.ActiveBuff{
			Effect:    cmd.Effect,
			ExpiresAt: ts.Add(cmd.Effect.Duration),
		}
		player.ActiveBuffs = append(player.ActiveBuffs, buff)
		player.LastUpdated = ts
		affected = append(affected, copyPlayer(player))
	}

	room.LastUpdated = ts
	stats.ActiveBuffHistogram[cmd.Effect.EffectID] += len(affected)
	stats.InteractionCount += len(affected)
	stats.LastUpdated = ts

	return domain.BuffResult{
		AffectedPlayers: affected,
		Room:            copyRoom(room),
		Stats:           copyRoomStats(stats),
	}, nil
}

func (s *Store) applyAttack(cmd domain.AttackCommand, ts time.Time) (domain.AttackResult, error) {
	attacker, err := s.getPlayer(cmd.AttackerID)
	if err != nil {
		return domain.AttackResult{}, err
	}
	target, err := s.getPlayer(cmd.TargetID)
	if err != nil {
		return domain.AttackResult{}, err
	}
	room, err := s.getRoom(cmd.RoomID)
	if err != nil {
		return domain.AttackResult{}, err
	}
	stats, err := s.statsForRoom(room)
	if err != nil {
		return domain.AttackResult{}, err
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
	stats.AverageEnergy = s.recomputeAverageEnergy(room)
	stats.LastUpdated = ts

	return domain.AttackResult{
		Attacker: copyPlayer(attacker),
		Target:   copyPlayer(target),
		Room:     copyRoom(room),
		Stats:    copyRoomStats(stats),
	}, nil
}

func (s *Store) applyTrade(cmd domain.TradeCommand, ts time.Time) (domain.TradeResult, error) {
	initiator, err := s.getPlayer(cmd.InitiatorID)
	if err != nil {
		return domain.TradeResult{}, err
	}
	partner, err := s.getPlayer(cmd.PartnerID)
	if err != nil {
		return domain.TradeResult{}, err
	}
	room, err := s.getRoom(cmd.RoomID)
	if err != nil {
		return domain.TradeResult{}, err
	}

	if err := applyInventoryChanges(initiator, cmd.Offer, -1); err != nil {
		return domain.TradeResult{}, err
	}
	if err := applyInventoryChanges(partner, cmd.Request, -1); err != nil {
		return domain.TradeResult{}, err
	}
	if err := applyInventoryChanges(initiator, cmd.Request, +1); err != nil {
		return domain.TradeResult{}, err
	}
	if err := applyInventoryChanges(partner, cmd.Offer, +1); err != nil {
		return domain.TradeResult{}, err
	}

	initiator.LastUpdated = ts
	partner.LastUpdated = ts

	room.LastUpdated = ts
	stats, err := s.statsForRoom(room)
	if err != nil {
		return domain.TradeResult{}, err
	}
	stats.InteractionCount++
	stats.LastUpdated = ts

	return domain.TradeResult{
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

func (s *Store) recomputeAverageEnergy(room *domain.RoomSnapshot) float64 {
	if len(room.MemberIDs) == 0 {
		return 0
	}
	sum := 0
	count := 0
	for _, id := range room.MemberIDs {
		if player, ok := s.players[id]; ok {
			sum += player.Energy
			count++
		}
	}
	if count == 0 {
		return 0
	}
	return float64(sum) / float64(count)
}

func clonePlayerPtr(src *domain.PlayerSnapshot) *domain.PlayerSnapshot {
	if src == nil {
		return nil
	}
	cp := *src
	cp.ActiveBuffs = append([]domain.ActiveBuff(nil), src.ActiveBuffs...)
	cp.Inventory = append([]domain.InventoryEntry(nil), src.Inventory...)
	return &cp
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

func cloneRoomPtr(src *domain.RoomSnapshot) *domain.RoomSnapshot {
	if src == nil {
		return nil
	}
	cp := *src
	cp.MemberIDs = append([]string(nil), src.MemberIDs...)
	return &cp
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
	stats.AverageEnergy = s.recomputeAverageEnergy(room)
	stats.LastUpdated = room.LastUpdated
	for _, id := range room.MemberIDs {
		if player, ok := s.players[id]; ok {
			for _, buff := range player.ActiveBuffs {
				stats.ActiveBuffHistogram[buff.Effect.EffectID]++
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
