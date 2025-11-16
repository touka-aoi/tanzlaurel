package utils

import (
	"math"

	"github.com/touka-aoi/paralle-vs-single/domain"
)

func FiniteVec(v domain.Vec2) bool {
	return isFinite(v.X) && isFinite(v.Y)
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
