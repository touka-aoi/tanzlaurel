package application

import (
	"testing"
)

func TestNewMap(t *testing.T) {
	m := NewMap(10, 20, 1.5)

	if m.Width != 10 {
		t.Errorf("Width = %d, want 10", m.Width)
	}
	if m.Height != 20 {
		t.Errorf("Height = %d, want 20", m.Height)
	}
	if m.TileSize != 1.5 {
		t.Errorf("TileSize = %f, want 1.5", m.TileSize)
	}
	if len(m.Tiles) != 200 {
		t.Errorf("Tiles length = %d, want 200", len(m.Tiles))
	}
}

func TestMap_GetSetTile(t *testing.T) {
	m := NewMap(10, 10, 1.0)

	// 初期値はTileEmpty
	if tile := m.GetTile(5, 5); tile != TileEmpty {
		t.Errorf("initial tile = %d, want TileEmpty", tile)
	}

	// タイルを設定
	if err := m.SetTile(5, 5, TileWall); err != nil {
		t.Fatalf("SetTile failed: %v", err)
	}
	if tile := m.GetTile(5, 5); tile != TileWall {
		t.Errorf("tile = %d, want TileWall", tile)
	}
}

func TestMap_GetTile_OutOfBounds(t *testing.T) {
	m := NewMap(10, 10, 1.0)

	// 範囲外はTileWallを返す
	cases := []struct {
		x, y int
	}{
		{-1, 5},
		{10, 5},
		{5, -1},
		{5, 10},
	}

	for _, c := range cases {
		if tile := m.GetTile(c.x, c.y); tile != TileWall {
			t.Errorf("GetTile(%d, %d) = %d, want TileWall", c.x, c.y, tile)
		}
	}
}

func TestMap_SetTile_OutOfBounds(t *testing.T) {
	m := NewMap(10, 10, 1.0)

	tests := []struct {
		name string
		x, y int
	}{
		{"negative x", -1, 5},
		{"x exceeds width", 10, 5},
		{"negative y", 5, -1},
		{"y exceeds height", 5, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := m.SetTile(tt.x, tt.y, TileWall)
			if err == nil {
				t.Errorf("expected error, got nil")
			}
			if _, ok := err.(*TileOutOfRangeError); !ok {
				t.Errorf("expected TileOutOfRangeError, got %T", err)
			}
		})
	}
}

func TestMap_WorldSize(t *testing.T) {
	m := NewMap(10, 20, 2.5)

	if w := m.WorldWidth(); w != 25.0 {
		t.Errorf("WorldWidth = %f, want 25.0", w)
	}
	if h := m.WorldHeight(); h != 50.0 {
		t.Errorf("WorldHeight = %f, want 50.0", h)
	}
}
