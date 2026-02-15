package application

import "withered/server/domain"

// BotAction はボットの行動を表します。
type BotAction struct {
	MoveDirection domain.Position2D
}

// BotController はボットの意思決定インターフェースです。
type BotController interface {
	Decide(self *Actor, allActors []*Actor, allBullets []*Bullet) BotAction
}

// BotInstance はボットのインスタンスを表します。
type BotInstance struct {
	SessionID domain.SessionID
}
