package service

import (
	"errors"
	"fmt"
	"math"

	"github.com/touka-aoi/paralle-vs-single/application/domain"
	"github.com/touka-aoi/paralle-vs-single/application/request"
)

// SimpleValidator は最低限の入力検証を提供するデフォルト実装。
type SimpleValidator struct{}

func (SimpleValidator) Move(req request.Move) error {
	cmd := req.Command
	if cmd.ActorID == "" {
		return errors.New("actor id is required")
	}
	if cmd.RoomID == "" {
		return errors.New("room id is required")
	}
	if !finiteVec(cmd.NextPosition) {
		return fmt.Errorf("invalid position: %+v", cmd.NextPosition)
	}
	return nil
}

func (SimpleValidator) Buff(req request.Buff) error {
	cmd := req.Command
	if cmd.CasterID == "" {
		return errors.New("caster id is required")
	}
	if cmd.RoomID == "" {
		return errors.New("room id is required")
	}
	if cmd.Effect.EffectID == "" {
		return errors.New("effect id is required")
	}
	if cmd.Effect.Duration <= 0 {
		return errors.New("duration must be positive")
	}
	return nil
}

func (SimpleValidator) Attack(req request.Attack) error {
	cmd := req.Command
	if cmd.AttackerID == "" || cmd.TargetID == "" {
		return errors.New("attacker and target ids are required")
	}
	if cmd.RoomID == "" {
		return errors.New("room id is required")
	}
	if cmd.Damage <= 0 {
		return errors.New("damage must be positive")
	}
	return nil
}

func (SimpleValidator) Trade(req request.Trade) error {
	cmd := req.Command
	if cmd.InitiatorID == "" || cmd.PartnerID == "" {
		return errors.New("initiator and partner ids are required")
	}
	if cmd.RoomID == "" {
		return errors.New("room id is required")
	}
	if len(cmd.Offer) == 0 && len(cmd.Request) == 0 {
		return errors.New("either offer or request must be present")
	}
	return nil
}

func finiteVec(v domain.Vec2) bool {
	return !(isFinite(v.X) || isFinite(v.Y))
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
