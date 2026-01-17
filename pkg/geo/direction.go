package geo

type Direction uint8

const (
	// Основные направления

	DirNone  Direction = iota
	DirNorth           // Y - 1
	DirSouth           // Y + 1
	DirWest            // X - 1
	DirEast            // X + 1
	DirUp              // Z + 1
	DirDown            // Z - 1

	// Диагонали

	DirNorthWest
	DirNorthEast
	DirSouthWest
	DirSouthEast
)

// Offset возвращает дельту координат для направления.
func (d Direction) Offset() (dx, dy, dz int) {
	switch d {
	case DirNorth:
		return 0, -1, 0
	case DirSouth:
		return 0, 1, 0
	case DirWest:
		return -1, 0, 0
	case DirEast:
		return 1, 0, 0
	case DirUp:
		return 0, 0, 1
	case DirDown:
		return 0, 0, -1
	case DirNorthWest:
		return -1, -1, 0
	case DirNorthEast:
		return 1, -1, 0
	case DirSouthWest:
		return -1, 1, 0
	case DirSouthEast:
		return 1, 1, 0
	case DirNone:
	default:
		return 0, 0, 0
	}
	return 0, 0, 0
}

// Move сдвигает Location в указанном направлении.
func (l Location) Move(d Direction) Location {
	dx, dy, dz := d.Offset()
	return l.Shift(dx, dy, dz)
}
