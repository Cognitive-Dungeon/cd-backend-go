package grid

// Neighbors4 содержит смещения к четырём ортогональным соседним тайлам:
//
//	(1, 0), (-1, 0), (0, 1), (0, -1)
//
// Используется для:
//   - 4-направленного движения
//   - flood fill
//   - Manhattan pathfinding
//   - симуляций без диагоналей
var Neighbors4 = [4]TilePos{
	{X: 1, Y: 0},
	{X: -1, Y: 0},
	{X: 0, Y: 1},
	{X: 0, Y: -1},
}

// Neighbors8 содержит смещения к восьми соседним тайлам,
// включая диагональные направления.
//
// Используется для:
//   - 8-направленного движения
//   - FOV
//   - A* с диагоналями
//   - симуляций, где диагональный шаг допустим
var Neighbors8 = [8]TilePos{
	{X: 1, Y: 0},
	{X: -1, Y: 0},
	{X: 0, Y: 1},
	{X: 0, Y: -1},
	{X: 1, Y: 1},
	{X: 1, Y: -1},
	{X: -1, Y: 1},
	{X: -1, Y: -1},
}

// NeighborsOrthogonal содержит только ортогональные смещения.
var NeighborsOrthogonal = Neighbors4

// NeighborsDiagonal содержит только диагональные смещения.
var NeighborsDiagonal = [4]TilePos{
	{X: 1, Y: 1},
	{X: 1, Y: -1},
	{X: -1, Y: 1},
	{X: -1, Y: -1},
}

// ForEachNeighbor4 вызывает fn для каждого ортогонального соседа позиции p.
func ForEachNeighbor4(p TilePos, fn func(TilePos)) {
	for _, d := range Neighbors4 {
		fn(p.Add(d))
	}
}

// ForEachNeighbor8 вызывает fn для каждого соседа (включая диагонали).
func ForEachNeighbor8(p TilePos, fn func(TilePos)) {
	for _, d := range Neighbors8 {
		fn(p.Add(d))
	}
}

// NeighborCost описывает смещение и стоимость шага.
type NeighborCost struct {
	Offset TilePos
	Cost   int32
}

// Neighbors4WithCost используется в pathfinding без диагоналей
var Neighbors4WithCost = [4]NeighborCost{
	{Offset: TilePos{X: 1, Y: 0}, Cost: 10},
	{Offset: TilePos{X: -1, Y: 0}, Cost: 10},
	{Offset: TilePos{X: 0, Y: 1}, Cost: 10},
	{Offset: TilePos{X: 0, Y: -1}, Cost: 10},
}

// Neighbors8WithCost используется в pathfinding с диагоналями.
var Neighbors8WithCost = [8]NeighborCost{
	{Offset: TilePos{X: 1, Y: 0}, Cost: 10},
	{Offset: TilePos{X: -1, Y: 0}, Cost: 10},
	{Offset: TilePos{X: 0, Y: 1}, Cost: 10},
	{Offset: TilePos{X: 0, Y: -1}, Cost: 10},
	{Offset: TilePos{X: 1, Y: 1}, Cost: 14},
	{Offset: TilePos{X: 1, Y: -1}, Cost: 14},
	{Offset: TilePos{X: -1, Y: 1}, Cost: 14},
	{Offset: TilePos{X: -1, Y: -1}, Cost: 14},
}

// ForEachNeighbor4WithCost вызывает fn для каждого ортогонального соседа
// позиции p, передавая позицию соседа и стоимость шага.
func ForEachNeighbor4WithCost(p TilePos, fn func(TilePos, int32)) {
	for _, n := range Neighbors4WithCost {
		fn(p.Add(n.Offset), n.Cost)
	}
}

// ForEachNeighbor8WithCost вызывает fn для каждого соседа (включая диагонали)
// позиции p, передавая позицию соседа и стоимость шага.
func ForEachNeighbor8WithCost(p TilePos, fn func(TilePos, int32)) {
	for _, n := range Neighbors8WithCost {
		fn(p.Add(n.Offset), n.Cost)
	}
}
