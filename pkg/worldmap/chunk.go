package worldmap

import (
	"cognitive-server/pkg/geo"
)

const (
	ChunkSize  = 16
	ChunkShift = 4  // 2^4 = 16
	ChunkMask  = 15 // 0xF (0000...1111)
	ChunkTotal = ChunkSize * ChunkSize
)

// Chunk — сегмент карты размером 16x16 тайлов.
type Chunk struct {
	// Плоский массив для лучшей локальности кэша процессора.
	// Индекс = y * 16 + x
	Tiles [ChunkTotal]Tile
}

// NewChunk создает новый пустой чанк.
func NewChunk() *Chunk {
	return &Chunk{}
}

// SetTile устанавливает тайл в локальных координатах (0-15).
// Возвращает false, если координаты выходят за границы.
func (c *Chunk) SetTile(lx, ly int, tile Tile) bool {
	if lx < 0 || lx >= ChunkSize || ly < 0 || ly >= ChunkSize {
		return false
	}
	// Бинарный сдвиг ly << 4 эквивалентен ly * 16, но быстрее/нагляднее для степеней двойки
	c.Tiles[(ly<<ChunkShift)|lx] = tile
	return true
}

// GetTile возвращает тайл по локальным координатам.
func (c *Chunk) GetTile(lx, ly int) (Tile, bool) {
	if lx < 0 || lx >= ChunkSize || ly < 0 || ly >= ChunkSize {
		return Tile{}, false
	}
	return c.Tiles[(ly<<ChunkShift)|lx], true
}

// --- Хелперы для координат ---

// GetChunkKey возвращает "координату чанка" для глобальной позиции.
// Пример: Pos(17, 0, 5) -> Pos(1, 0, 5).
// Используется как ключ в map[geo.Location]*Chunk.
func GetChunkKey(pos geo.Location) geo.Location {
	x, y, z := pos.XYZ()
	// Сдвигаем биты вправо, получая индекс чанка
	return geo.Pos(x>>ChunkShift, y>>ChunkShift, z)
}

// GetLocalCoords возвращает координаты внутри чанка (0-15) для глобальной позиции.
func GetLocalCoords(pos geo.Location) (lx, ly int) {
	x, y, _ := pos.XYZ()
	// Берем младшие 4 бита
	return x & ChunkMask, y & ChunkMask
}
