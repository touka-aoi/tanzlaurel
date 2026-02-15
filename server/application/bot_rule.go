package application

import (
	"math"
	"math/rand/v2"

	"withered/server/domain"
)

const (
	botDangerDist float32 = 3.0  // 弾丸回避を始める距離
	botNoiseAngle float64 = 0.52 // ±30度 (π/6 ≈ 0.52 rad)
	rushChance    float64 = 0.02 // 毎tick 2% の確率で突撃
)

// RuleBotController はルールベースのボットAIです。
// ボットごとに異なる個性パラメータを持ちます。
type RuleBotController struct {
	CloseRange float32 // 後退を始める距離
	MidRange   float32 // ストレイフを始める距離
	StrafeSign float32 // +1: 反時計回り, -1: 時計回り
}

// NewRuleBotController はランダムな個性を持つボットAIを生成します。
func NewRuleBotController() *RuleBotController {
	strafeSign := float32(1.0)
	if rand.Float64() < 0.5 {
		strafeSign = -1.0
	}
	return &RuleBotController{
		CloseRange: 3.0 + rand.Float32()*4.0,  // 3〜7
		MidRange:   10.0 + rand.Float32()*10.0, // 10〜20
		StrafeSign: strafeSign,
	}
}

func (r *RuleBotController) Decide(self *Actor, allActors []*Actor, allBullets []*Bullet) BotAction {
	// 被弾回避を優先
	if dir, ok := r.evadeBullet(self, allBullets); ok {
		return BotAction{MoveDirection: addNoise(dir)}
	}

	// 最寄り敵に対する行動
	nearest := r.findNearestEnemy(self, allActors)
	if nearest == nil {
		return BotAction{}
	}

	dx := nearest.Position.X - self.Position.X
	dy := nearest.Position.Y - self.Position.Y
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))
	if dist < 0.001 {
		return BotAction{}
	}

	// 正規化
	nx := dx / dist
	ny := dy / dist

	// ランダム突撃: 一定確率で距離に関係なく接近
	if rand.Float64() < rushChance {
		return BotAction{MoveDirection: addNoise(domain.Position2D{X: nx, Y: ny})}
	}

	var dir domain.Position2D
	switch {
	case dist < r.CloseRange:
		// 近距離: 後退
		dir = domain.Position2D{X: -nx, Y: -ny}
	case dist < r.MidRange:
		// 中距離: 横移動（ストレイフ方向はボットごとに異なる）
		dir = domain.Position2D{X: -ny * r.StrafeSign, Y: nx * r.StrafeSign}
	default:
		// 遠距離: 接近
		dir = domain.Position2D{X: nx, Y: ny}
	}

	return BotAction{MoveDirection: addNoise(dir)}
}

// evadeBullet は自分に向かってくる弾丸を回避する方向を返します。
func (r *RuleBotController) evadeBullet(self *Actor, bullets []*Bullet) (domain.Position2D, bool) {
	var closestDist float32 = math.MaxFloat32
	var closestBullet *Bullet

	for _, b := range bullets {
		if b.OwnerID == self.SessionID {
			continue
		}
		dx := self.Position.X - b.Position.X
		dy := self.Position.Y - b.Position.Y
		dist := float32(math.Sqrt(float64(dx*dx + dy*dy)))

		if dist > botDangerDist {
			continue
		}

		// 弾丸が自分に向かっているか確認（内積 > 0）
		dot := dx*b.Velocity.X + dy*b.Velocity.Y
		if dot <= 0 {
			continue
		}

		if dist < closestDist {
			closestDist = dist
			closestBullet = b
		}
	}

	if closestBullet == nil {
		return domain.Position2D{}, false
	}

	// 弾丸の進行方向に対して垂直に回避
	vLen := float32(math.Sqrt(float64(closestBullet.Velocity.X*closestBullet.Velocity.X + closestBullet.Velocity.Y*closestBullet.Velocity.Y)))
	if vLen < 0.001 {
		return domain.Position2D{}, false
	}
	return domain.Position2D{
		X: -closestBullet.Velocity.Y / vLen,
		Y: closestBullet.Velocity.X / vLen,
	}, true
}

// findNearestEnemy は最寄りの生存敵を探します。
func (r *RuleBotController) findNearestEnemy(self *Actor, allActors []*Actor) *Actor {
	var nearest *Actor
	var nearestDistSq float32 = math.MaxFloat32

	for _, other := range allActors {
		if other.SessionID == self.SessionID || !other.IsAlive() {
			continue
		}
		dx := other.Position.X - self.Position.X
		dy := other.Position.Y - self.Position.Y
		distSq := dx*dx + dy*dy
		if distSq < nearestDistSq {
			nearestDistSq = distSq
			nearest = other
		}
	}
	return nearest
}

// addNoise は移動方向に ±30度 のランダムノイズを加えます。
func addNoise(dir domain.Position2D) domain.Position2D {
	noise := (rand.Float64()*2 - 1) * botNoiseAngle
	cos := float32(math.Cos(noise))
	sin := float32(math.Sin(noise))
	return domain.Position2D{
		X: dir.X*cos - dir.Y*sin,
		Y: dir.X*sin + dir.Y*cos,
	}
}
