package enums

type Tile uint8

const (
	TileFloor Tile = iota
	TileWall
)

type Direction int8

const (
	DirNone Direction = iota
	DirUp
	DirDown
	DirLeft
	DirRight
)
