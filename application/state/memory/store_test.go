package memory

import (
	"testing"
	"time"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
)

func TestStoreApplyMove(t *testing.T) {
	store := newTestStore()

	result, err := store.applyMove(sampleMoveCommand("player-1"), time.Unix(100, 0))
	if err != nil {
		t.Fatalf("applyMove returned error: %v", err)
	}

	if result.Player.Position != (domain.Vec2{X: 10, Y: 20}) {
		t.Errorf("unexpected position: %+v", result.Player.Position)
	}

	if result.Player.LastUpdated != time.Unix(100, 0) {
		t.Errorf("expected LastUpdated to be set, got %v", result.Player.LastUpdated)
	}
	if result.Player.PlayerID != "player-1" {
		t.Errorf("unexpected player id: %s", result.Player.PlayerID)
	}

	stats := store.roomStats["room-1"]
	if stats.TotalHealth != 200 {
		t.Errorf("expected total health to remain 200, got %d", stats.TotalHealth)
	}
}

func TestStoreApplyBuff(t *testing.T) {
	store := newTestStore()

	buffCmd := domain.BuffCommand{
		CasterID:  "player-1",
		RoomID:    "room-1",
		TargetIDs: []string{"player-1", "player-2"},
		Effect: domain.BuffEffect{
			EffectID:  "atk-up",
			Magnitude: 1.2,
			Duration:  time.Minute,
		},
	}

	result, err := store.applyBuff(buffCmd, time.Unix(200, 0))
	if err != nil {
		t.Fatalf("applyBuff returned error: %v", err)
	}

	if len(result.AffectedPlayers) != 2 {
		t.Fatalf("expected 2 affected players, got %d", len(result.AffectedPlayers))
	}

	stats := store.roomStats["room-1"]
	if stats.ActiveBuffHistogram["atk-up"] != 2 {
		t.Errorf("expected buff histogram to count 2, got %d", stats.ActiveBuffHistogram["atk-up"])
	}
	if stats.InteractionCount != 2 {
		t.Errorf("expected interaction count 2, got %d", stats.InteractionCount)
	}
	if stats.LastUpdated != time.Unix(200, 0) {
		t.Errorf("expected stats LastUpdated to be set, got %v", stats.LastUpdated)
	}
}

func TestStoreApplyAttack(t *testing.T) {
	store := newTestStore()

	attackCmd := domain.AttackCommand{
		AttackerID: "player-1",
		TargetID:   "player-2",
		RoomID:     "room-1",
		Damage:     30,
	}

	result, err := store.applyAttack(attackCmd, time.Unix(300, 0))
	if err != nil {
		t.Fatalf("applyAttack returned error: %v", err)
	}

	if result.Target.Health != 70 {
		t.Errorf("expected target health 70 after attack, got %d", result.Target.Health)
	}

	stats := store.roomStats["room-1"]
	if stats.TotalHealth != 170 {
		t.Errorf("expected total health 170, got %d", stats.TotalHealth)
	}
	if stats.InteractionCount != 1 {
		t.Errorf("expected interaction count 1, got %d", stats.InteractionCount)
	}
	if stats.LastUpdated != time.Unix(300, 0) {
		t.Errorf("expected stats last updated at 300, got %v", stats.LastUpdated)
	}
}

func TestStoreApplyTrade(t *testing.T) {
	store := newTestStore()

	tradeCmd := domain.TradeCommand{
		InitiatorID: "player-1",
		PartnerID:   "player-2",
		RoomID:      "room-1",
		Offer: []domain.ItemChange{
			{ItemID: "gold", QuantityDelta: 10},
		},
		Request: []domain.ItemChange{
			{ItemID: "potion", QuantityDelta: 2},
		},
	}

	result, err := store.applyTrade(tradeCmd, time.Unix(400, 0))
	if err != nil {
		t.Fatalf("applyTrade returned error: %v", err)
	}

	if result.Ledger.Confirmed != true {
		t.Fatalf("expected ledger confirmed")
	}

	initiator := store.players["player-1"]
	if !hasInventoryItem(initiator.Inventory, "gold", 40) {
		t.Errorf("expected player-1 gold 40, inventory=%+v", initiator.Inventory)
	}
	if !hasInventoryItem(initiator.Inventory, "potion", 7) {
		t.Errorf("expected player-1 potion 7, inventory=%+v", initiator.Inventory)
	}

	partner := store.players["player-2"]
	if !hasInventoryItem(partner.Inventory, "gold", 60) {
		t.Errorf("expected player-2 gold 60, inventory=%+v", partner.Inventory)
	}
	if !hasInventoryItem(partner.Inventory, "potion", 3) {
		t.Errorf("expected player-2 potion 3, inventory=%+v", partner.Inventory)
	}

	stats := store.roomStats["room-1"]
	if stats.InteractionCount != 1 {
		t.Errorf("expected interaction count 1, got %d", stats.InteractionCount)
	}
	if stats.LastUpdated != time.Unix(400, 0) {
		t.Errorf("expected stats last updated at 400, got %v", stats.LastUpdated)
	}
}

func TestStoreApplyTradeUnderflow(t *testing.T) {
	store := newTestStore()

	tradeCmd := domain.TradeCommand{
		InitiatorID: "player-1",
		PartnerID:   "player-2",
		RoomID:      "room-1",
		Offer: []domain.ItemChange{
			{ItemID: "potion", QuantityDelta: 1000},
		},
	}

	_, err := store.applyTrade(tradeCmd, time.Unix(500, 0))
	if err == nil {
		t.Fatalf("expected error for inventory underflow")
	}
}

func newTestStore() *Store {
	players := []domain.PlayerSnapshot{
		{
			PlayerID: "player-1",
			RoomID:   "room-1",
			Position: domain.Vec2{X: 0, Y: 0},
			Health:   100,
			Energy:   50,
			Inventory: []domain.InventoryEntry{
				{ItemID: "gold", Quantity: 50},
				{ItemID: "potion", Quantity: 5},
			},
			LastUpdated: time.Unix(0, 0),
		},
		{
			PlayerID: "player-2",
			RoomID:   "room-1",
			Position: domain.Vec2{X: 5, Y: 5},
			Health:   100,
			Energy:   40,
			Inventory: []domain.InventoryEntry{
				{ItemID: "gold", Quantity: 50},
				{ItemID: "potion", Quantity: 5},
			},
			LastUpdated: time.Unix(0, 0),
		},
	}

	rooms := []domain.RoomSnapshot{
		{
			RoomID:      "room-1",
			MemberIDs:   []string{"player-1", "player-2"},
			LastUpdated: time.Unix(0, 0),
		},
	}

	return NewStore(players, rooms)
}

func sampleMoveCommand(playerID string) domain.MoveCommand {
	return domain.MoveCommand{
		ActorID:      playerID,
		RoomID:       "room-1",
		NextPosition: domain.Vec2{X: 10, Y: 20},
		Facing:       1.0,
	}
}

func hasInventoryItem(inv []domain.InventoryEntry, itemID string, quantity int) bool {
	for _, entry := range inv {
		if entry.ItemID == itemID && entry.Quantity == quantity {
			return true
		}
	}
	return false
}
