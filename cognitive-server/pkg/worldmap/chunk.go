package worldmap

import (
	"cognitive-server/pkg/geo"
)

const (
	ChunkSize  = 16
	ChunkShift = 4  // 2^4 = 16
	ChunkMask  = 15 // 0xF (0000...1111)
	ChunkArea  = ChunkSize * ChunkSize
)

// Chunk — статический сегмент карты размером 16x16 тайлов.
type Chunk struct {
	// Indices хранит индексы палитры.
	Indices [ChunkArea]uint8

	// Palette хранит уникальные тайлы
	Palette []Tile

	// Кешированные маски для быстрого доступа
	SolidMask  [4]uint64
	OpaqueMask [4]uint64
}

// NewChunk создает новый пустой чанк.
// Где каждый тайл с 0-ым индексом что = Void
func NewChunk() *Chunk {
	return &Chunk{
		Palette: []Tile{{}},
	}
}

// SetTile устанавливает тайл в локальных координатах (0-15) обновляя палитру и маски.
// Возвращает false, если координаты выходят за границы.
func (c *Chunk) SetTile(lx, ly int, tile Tile) bool {
	if uint(lx) >= ChunkSize || uint(ly) >= ChunkSize {
		return false
	}
	// 1. Сначала ищем, существует ли уже тайл в палитре.
	packedTile := tile.Pack()
	palIdx := -1
	for i, t := range c.Palette {
		if t.Pack() == packedTile {
			palIdx = i
			break
		}
	}
	// 2. Тайл в палитре не найден, добавляем
	if palIdx == -1 {
		if len(c.Palette) >= ChunkArea {
			return false
		}
		c.Palette = append(c.Palette, tile)
		palIdx = len(c.Palette) - 1
	}

	// 3. Записываем новый тайл в индекс
	flatIdx := (ly << ChunkShift) | lx
	c.Indices[flatIdx] = uint8(palIdx)

	// 4. Обновляем маски для тайла
	c.updateSingleMaskBit(flatIdx, tile)

	return true
}

// GetTile возвращает тайл из статики по локальным координатам.
// Если тайл найден возвращает тайл и true
// Если тайл не найден возвращает пустой тайл и false
func (c *Chunk) GetTile(lx, ly int) (Tile, bool) {
	if lx < 0 || lx >= ChunkSize || ly < 0 || ly >= ChunkSize {
		return Tile{}, false
	}
	idx := c.Indices[(ly<<ChunkShift)|lx]
	return c.Palette[idx], true
}

// RebuildMasks полностью пересчитывает маски (после загрузки/дедупликации).
func (c *Chunk) RebuildMasks() {
	c.SolidMask = [4]uint64{}
	c.OpaqueMask = [4]uint64{}

	// Кэш свойств палитры
	pSolid := make([]bool, len(c.Palette))
	pOpaque := make([]bool, len(c.Palette))
	for i, t := range c.Palette {
		pSolid[i] = t.Flags.Has(FlagSolid)
		pOpaque[i] = t.Flags.Has(FlagOpaque)
	}

	for i, palIdx := range c.Indices {
		if int(palIdx) >= len(c.Palette) {
			continue
		}
		if pSolid[palIdx] {
			c.setBit(i, &c.SolidMask)
		}
		if pOpaque[palIdx] {
			c.setBit(i, &c.OpaqueMask)
		}
	}
}

func (c *Chunk) IsSolidLocal(lx, ly int) bool {
	if uint(lx) >= ChunkSize || uint(ly) >= ChunkSize {
		return false
	}
	return c.checkBit((ly<<ChunkShift)|lx, &c.SolidMask)
}

func (c *Chunk) IsOpaqueLocal(lx, ly int) bool {
	if uint(lx) >= ChunkSize || uint(ly) >= ChunkSize {
		return false
	}
	return c.checkBit((ly<<ChunkShift)|lx, &c.OpaqueMask)
}

// --- Dynamic Layer (Delta) ---

// SparseChunk хранит изменения поверх статичного Chunk.
type SparseChunk struct {
	Modifications map[uint8]Tile
	IsDirty       bool

	// Кэш масок (Static + Delta).
	SolidMask  [4]uint64 `json:"-"`
	OpaqueMask [4]uint64 `json:"-"`
}

func NewSparseChunk() *SparseChunk {
	return &SparseChunk{
		Modifications: make(map[uint8]Tile),
	}
}

// UpdateMasks синхронизирует маски со статическим хранилищем
func (sc *SparseChunk) UpdateMasks(base *Chunk) {
	// 1. Копируем базу
	if base != nil {
		sc.SolidMask = base.SolidMask
		sc.OpaqueMask = base.OpaqueMask
	} else {
		sc.SolidMask = [4]uint64{}
		sc.OpaqueMask = [4]uint64{}
	}

	// 2. Накатываем изменения
	for idx, tile := range sc.Modifications {
		sc.updateSingleMaskBit(int(idx), tile)
	}
}

func (sc *SparseChunk) Get(lx, ly int) (Tile, bool) {
	idx := uint8((ly << ChunkShift) | lx)
	t, ok := sc.Modifications[idx]
	return t, ok
}

func (sc *SparseChunk) Set(lx, ly int, t Tile) {
	idx := uint8((ly << ChunkShift) | lx)
	sc.Modifications[idx] = t
	sc.IsDirty = true
	sc.updateSingleMaskBit(int(idx), t)
}

// --- Bitwise Helpers ---

func (c *Chunk) setBit(idx int, mask *[4]uint64) {
	mask[idx>>6] |= 1 << (idx & 63)
}

func (c *Chunk) checkBit(idx int, mask *[4]uint64) bool {
	return (mask[idx>>6] & (1 << (idx & 63))) != 0
}

func (c *Chunk) updateSingleMaskBit(idx int, tile Tile) {
	block := idx >> 6
	bit := uint64(1) << (idx & 63)

	if tile.Flags.Has(FlagSolid) {
		c.SolidMask[block] |= bit
	} else {
		c.SolidMask[block] &^= bit
	}
	if tile.Flags.Has(FlagOpaque) {
		c.OpaqueMask[block] |= bit
	} else {
		c.OpaqueMask[block] &^= bit
	}
}

func (sc *SparseChunk) updateSingleMaskBit(idx int, tile Tile) {
	block := idx >> 6
	bit := uint64(1) << (idx & 63)

	if tile.Flags.Has(FlagSolid) {
		sc.SolidMask[block] |= bit
	} else {
		sc.SolidMask[block] &^= bit
	}
	if tile.Flags.Has(FlagOpaque) {
		sc.OpaqueMask[block] |= bit
	} else {
		sc.OpaqueMask[block] &^= bit
	}
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
