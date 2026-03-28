package application

import "fmt"

// TileOutOfRangeError はタイル座標が範囲外の場合のエラーです。
type TileOutOfRangeError struct {
	X, Y          int
	Width, Height int
}

func (e *TileOutOfRangeError) Error() string {
	return fmt.Sprintf("tile coordinates (%d, %d) out of range [0-%d, 0-%d]", e.X, e.Y, e.Width-1, e.Height-1)
}

type TileID uint8

const (
	TileEmpty TileID = iota
	TileWall
	TileWater
)

// Map はタイル状の2Dマップを表す構造体です。
type Map struct {
	Width    int
	Height   int
	TileSize float32
	Tiles    []TileID
}

// NewMap は指定サイズのマップを作成します。全タイルはTileEmptyで初期化されます。
func NewMap(width, height int, tileSize float32) *Map {
	return &Map{
		Width:    width,
		Height:   height,
		TileSize: tileSize,
		Tiles:    make([]TileID, width*height),
	}
}

// GetTile は指定座標のタイルを取得します。範囲外の場合はTileWallを返します。
func (m *Map) GetTile(x, y int) TileID {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return TileWall
	}
	return m.Tiles[y*m.Width+x]
}

// SetTile は指定座標にタイルを設定します。範囲外の場合はエラーを返します。
func (m *Map) SetTile(x, y int, tile TileID) error {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return &TileOutOfRangeError{X: x, Y: y, Width: m.Width, Height: m.Height}
	}
	m.Tiles[y*m.Width+x] = tile
	return nil
}

// WorldWidth はワールド座標系での幅を返します。
func (m *Map) WorldWidth() float32 {
	return float32(m.Width) * m.TileSize
}

// WorldHeight はワールド座標系での高さを返します。
func (m *Map) WorldHeight() float32 {
	return float32(m.Height) * m.TileSize
}
