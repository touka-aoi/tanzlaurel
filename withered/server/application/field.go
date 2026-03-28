package application

// Static は固定的なフィールド情報を保持する構造体です。
type Static struct {
	Map *Map
}

// NewField は指定されたマップでフィールドを作成します。
func NewField(m *Map) *Static {
	return &Static{
		Map: m,
	}
}

func clamp(v, min, max float32) float32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
