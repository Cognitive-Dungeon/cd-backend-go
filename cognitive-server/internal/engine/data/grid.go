package data

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/grid"
)

type (
	Tile      = enums.Tile
	TileCoord = grid.TileCoord
	TilePos   = grid.TilePos
)

// Grid — физическое представление уровня (стены, пол).
type Grid struct {
	Width  TileCoord
	Height TileCoord
	// Плоский массив: index = y * width + x
	Tiles []Tile
}

func NewGrid(w, h TileCoord) *Grid {
	size := int(w) * int(h)
	return &Grid{
		Width:  w,
		Height: h,
		Tiles:  make([]Tile, size),
	}
}

func (g *Grid) InBounds(pos TilePos) bool {
	return pos.X >= 0 && pos.X < g.Width &&
		pos.Y >= 0 && pos.Y < g.Height
}

func (g *Grid) index(x, y TileCoord) int {
	return int(y*g.Width + x)
}

func (g *Grid) IsWalkable(pos TilePos) bool {
	if !g.InBounds(pos) {
		return false
	}
	return g.Tiles[g.index(pos.X, pos.Y)] == enums.TileFloor
}

func (g *Grid) SetTile(pos TilePos, tile Tile) {
	if g.InBounds(pos) {
		g.Tiles[g.index(pos.X, pos.Y)] = tile
	}
}
