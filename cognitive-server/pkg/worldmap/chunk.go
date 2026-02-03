package worldmap

import (
	"cognitive-server/pkg/geo"
)

const (
	// ChunkSize — размер чанка по одной оси (16x16 тайлов).
	ChunkSize = 16

	// ChunkShift используется для быстрого умножения/деления на размер чанка:
	// 2^4 = 16.
	ChunkShift = 4

	// ChunkMask используется для получения локальных координат
	// внутри чанка (младшие 4 бита).
	ChunkMask = 15 // 0xF (0000...1111)

	// ChunkArea — общее количество тайлов в чанке.
	// 16 * 16 = 256.
	ChunkArea = ChunkSize * ChunkSize
)

// Chunk — статический сегмент карты размером 16x16 тайлов.
//
// Chunk является частью Static Layer и хранится внутри Region.
//
// Ключевые свойства:
//   - после загрузки считается immutable
//   - оптимизирован для быстрого чтения
//   - использует локальную палитру и битовые маски
//
// Любые изменения карты во время выполнения
// должны производиться ТОЛЬКО через SparseChunk.
type Chunk struct {
	// Indices хранит индексы локальной палитры.
	//
	// Каждый элемент — uint8, указывающий на индекс в Palette.
	// Размер фиксированный: 256 элементов (16x16).
	Indices [ChunkArea]uint8

	// Palette хранит уникальные тайлы, используемые в данном чанке.
	//
	// Palette[0] всегда зарезервирован под "пустой" тайл (Void).
	Palette []Tile

	// SolidMask и OpaqueMask — кешированные битовые маски
	// для быстрого определения свойств тайлов.
	//
	// Формат:
	//   - 256 бит = 4 * uint64
	//   - бит = 1 → свойство активно
	SolidMask  [4]uint64
	OpaqueMask [4]uint64
}

// NewChunk создает новый пустой чанк.
//
// Инварианты:
//   - Palette[0] всегда указывает на "пустой" тайл (Void)
//   - все Indices инициализированы нулями и указывают на Palette[0]
func NewChunk() *Chunk {
	return &Chunk{
		Palette: []Tile{{}},
	}
}

// SetTile устанавливает тайл по локальным координатам чанка.
//
// Используется ТОЛЬКО:
//   - генераторами карты
//   - загрузчиками мира
//
// ВАЖНО:
//   - не предназначен для runtime-изменений
//   - обновляет палитру, индексы и битовые маски
//
// Возвращает false, если координаты выходят за границы
// или палитра переполнена.
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
			// Защита от патологического случая:
			// 256 уникальных тайлов в одном чанке.
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

// GetTile возвращает тайл из статического чанка по локальным координатам.
//
// Возвращает:
//   - (Tile, true)  — если координаты валидны
//   - (Tile{}, false) — если координаты вне чанка
//
// Метод относительно "медленный", так как обращается к палитре.
// Для быстрых проверок следует использовать IsSolidLocal / IsOpaqueLocal.
func (c *Chunk) GetTile(lx, ly int) (Tile, bool) {
	if lx < 0 || lx >= ChunkSize || ly < 0 || ly >= ChunkSize {
		return Tile{}, false
	}
	idx := c.Indices[(ly<<ChunkShift)|lx]
	return c.Palette[idx], true
}

// RebuildMasks полностью пересчитывает битовые маски чанка.
//
// Используется:
//   - после загрузки чанка из внешнего источника
//   - после дедупликации палитры
//
// Bitmask считается кешем, а не источником истины.
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

// IsSolidLocal проверяет, блокирует ли тайл движение.
//
// Быстрый метод:
//   - не обращается к Tile или Palette
//   - использует только битовые маски
//
// Возвращает false для координат вне чанка.
func (c *Chunk) IsSolidLocal(lx, ly int) bool {
	if uint(lx) >= ChunkSize || uint(ly) >= ChunkSize {
		return false
	}
	return c.checkBit((ly<<ChunkShift)|lx, &c.SolidMask)
}

// IsOpaqueLocal проверяет, блокирует ли тайл свет (FOV).
//
// Аналогичен IsSolidLocal, но использует OpaqueMask.
func (c *Chunk) IsOpaqueLocal(lx, ly int) bool {
	if uint(lx) >= ChunkSize || uint(ly) >= ChunkSize {
		return false
	}
	return c.checkBit((ly<<ChunkShift)|lx, &c.OpaqueMask)
}

// --- Dynamic Layer (Delta) ---

// SparseChunk хранит изменения поверх статического Chunk.
//
// Используется в Dynamic Layer и хранится в Shard.
//
// Содержит:
//   - точечные изменения тайлов
//   - собственные битовые маски (Static + Delta)
type SparseChunk struct {
	// Modifications содержит изменённые тайлы.
	// Ключ — упакованный индекс (ly<<4 | lx).
	Modifications map[uint8]Tile

	// IsDirty помечает наличие несохранённых изменений.
	IsDirty bool

	// SolidMask / OpaqueMask — результирующие маски
	// с учётом статики и дельты.
	//
	// Не сериализуются.
	SolidMask  [4]uint64 `json:"-"`
	OpaqueMask [4]uint64 `json:"-"`
}

// NewSparseChunk создаёт новый пустой SparseChunk.
func NewSparseChunk() *SparseChunk {
	return &SparseChunk{
		Modifications: make(map[uint8]Tile),
	}
}

// UpdateMasks синхронизирует маски SparseChunk
// с базовым статическим Chunk.
//
// Алгоритм:
//  1. Копируем маски из базового Chunk
//  2. Накатываем изменения из Modifications
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

// Get возвращает изменённый тайл, если он существует.
func (sc *SparseChunk) Get(lx, ly int) (Tile, bool) {
	idx := uint8((ly << ChunkShift) | lx)
	t, ok := sc.Modifications[idx]
	return t, ok
}

// Set записывает изменение тайла и обновляет маски.
//
// Используется для runtime-изменений карты.
func (sc *SparseChunk) Set(lx, ly int, t Tile) {
	idx := uint8((ly << ChunkShift) | lx)
	sc.Modifications[idx] = t
	sc.IsDirty = true
	sc.updateSingleMaskBit(int(idx), t)
}

// --- Bitwise Helpers ---

// setBit устанавливает бит в маске по линейному индексу тайла.
func (c *Chunk) setBit(idx int, mask *[4]uint64) {
	mask[idx>>6] |= 1 << (idx & 63)
}

// checkBit проверяет установлен ли бит в маске.
func (c *Chunk) checkBit(idx int, mask *[4]uint64) bool {
	return (mask[idx>>6] & (1 << (idx & 63))) != 0
}

// updateSingleMaskBit обновляет битовые маски для одного тайла.
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

// updateSingleMaskBit обновляет битовые маски для одного тайла.
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

// GetChunkKey возвращает координату чанка для глобальной позиции.
//
// Пример:
//
//	Pos(17, 0, 5) → Pos(1, 0, 5)
//
// Используется как ключ в map[geo.Location]*Chunk.
func GetChunkKey(pos geo.Location) geo.Location {
	x, y, z := pos.XYZ()
	// Сдвигаем биты вправо, получая индекс чанка
	return geo.Pos(x>>ChunkShift, y>>ChunkShift, z)
}

// GetLocalCoords возвращает локальные координаты внутри чанка (0..15) для глобальной позиции.
func GetLocalCoords(pos geo.Location) (lx, ly int) {
	x, y, _ := pos.XYZ()
	// Берем младшие 4 бита
	return x & ChunkMask, y & ChunkMask
}
