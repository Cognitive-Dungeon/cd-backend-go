package grid

type TileCoord int32

type TilePos struct {
	X TileCoord
	Y TileCoord
}

func (a TilePos) Add(d TilePos) TilePos {
	return TilePos{
		X: a.X + d.X,
		Y: a.Y + d.Y,
	}
}

func (a TilePos) Sub(o TilePos) TilePos {
	return TilePos{
		X: a.X - o.X,
		Y: a.Y - o.Y,
	}
}

var Zero = TilePos{0, 0}
