package handler

import "github.com/touka-aoi/paralle-vs-single/application/domain"

type MovePayload struct {
	Meta    domain.Meta
	Command domain.MoveCommand
}

type BuffPayload struct {
	Meta    domain.Meta
	Command domain.BuffCommand
}

type AttackPayload struct {
	Meta    domain.Meta
	Command domain.AttackCommand
}

type TradePayload struct {
	Meta    domain.Meta
	Command domain.TradeCommand
}
