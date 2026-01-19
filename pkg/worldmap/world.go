package worldmap

import (
	"cognitive-server/pkg/geo"
	"sync"
)

// chunkKey - внутренний ключ для мапы (X, Y, Z упакованные).
type chunkKey uint64

// TileInfo — расширенная информация о тайле для внешних систем.
// Возвращается методами анализа (GetContext).
type TileInfo struct {
	Pos      geo.Location
	Material MaterialID
	Flags    TileFlag
}

// IsSolid — удобный хелпер для проверки проходимости.
func (ti TileInfo) IsSolid() bool {
	return ti.Flags.Has(FlagSolid)
}

// IsOpaque — удобный хелпер для проверки прозрачности.
func (ti TileInfo) IsOpaque() bool {
	return ti.Flags.Has(FlagOpaque)
}

// World — потокобезопасный контейнер для бесконечного 3D мира.
type World struct {
	// RWMutex обязателен, так как сервер многопоточный:
	// AI читает карту, Генератор пишет новые чанки, Игроки меняют тайлы.
	mu     sync.RWMutex
	chunks map[geo.Location]*Chunk

	// DefaultTile возвращается, если чанка не существует.
	// Обычно это "Пустота" (Material: 0) или "Коренная порода".
	DefaultTile Tile
}

// NewWorld создает новый пустой мир.
func NewWorld() *World {
	return &World{
		chunks: make(map[geo.Location]*Chunk),
	}
}

// SetTile устанавливает тайл по глобальным координатам.
// Создает чанк, если его еще нет.
func (w *World) SetTile(pos geo.Location, tile Tile) {
	chunkKey := GetChunkKey(pos)
	lx, ly := GetLocalCoords(pos)

	w.mu.Lock()
	defer w.mu.Unlock()

	chunk, exists := w.chunks[chunkKey]
	if !exists {
		chunk = NewChunk()
		w.chunks[chunkKey] = chunk
	}

	chunk.SetTile(lx, ly, tile)
}

// GetTile возвращает тайл по глобальным координатам.
// Если чанк не загружен или не существует, возвращает World.DefaultTile.
func (w *World) GetTile(pos geo.Location) Tile {
	chunkKey := GetChunkKey(pos)
	lx, ly := GetLocalCoords(pos)

	w.mu.RLock()
	defer w.mu.RUnlock()

	chunk, exists := w.chunks[chunkKey]
	if !exists {
		return w.DefaultTile
	}

	// Ошибки локальных координат тут быть не должно из-за маски,
	// но метод GetTile внутри чанка безопасен.
	t, ok := chunk.GetTile(lx, ly)
	if !ok {
		return w.DefaultTile
	}
	return t
}

// GetInfo возвращает структуру с информацией о тайле.
// Это основной метод для Pathfinding и AI.
func (w *World) GetInfo(pos geo.Location) TileInfo {
	tile := w.GetTile(pos)
	return TileInfo{
		Pos:      pos,
		Material: tile.Material,
		Flags:    tile.Flags,
	}
}

// Reset очищает весь мир (полезно для тестов или перезагрузки).
func (w *World) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for k := range w.chunks {
		delete(w.chunks, k)
	}
}

// PutChunk позволяет загрузить готовый чанк целиком.
// Это основной метод для генераторов и загрузчиков карт.
// Он заменяет существующий чанк, если таковой был.
func (w *World) PutChunk(pos geo.Location, c *Chunk) {
	// Нормализуем ключ (на случай, если передали координаты тайла, а не чанка)
	chunkKey := GetChunkKey(pos)

	w.mu.Lock()
	defer w.mu.Unlock()

	w.chunks[chunkKey] = c
}

// GetChunk возвращает указатель на чанк (или nil).
// ВНИМАНИЕ: Возвращает прямой указатель. Изменять чанк напрямую
// небезопасно без внешней синхронизации, если игра уже запущена.
// Используйте для чтения или сериализации.
func (w *World) GetChunk(pos geo.Location) *Chunk {
	chunkKey := GetChunkKey(pos)

	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.chunks[chunkKey]
}

// RangeChunks итерируется по всем загруженным чанкам.
// Если f возвращает false, итерация прекращается.
func (w *World) RangeChunks(f func(pos geo.Location, c *Chunk) bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for pos, c := range w.chunks {
		if !f(pos, c) {
			break
		}
	}
}

// Bounds возвращает границы загруженного мира (Min, Max в координатах чанков).
// Полезно для рендера всей карты или миникарты.
func (w *World) Bounds() (min, max geo.Location) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	first := true
	var minX, minY, minZ int
	var maxX, maxY, maxZ int

	for pos := range w.chunks {
		x, y, z := pos.XYZ()
		if first {
			minX, minY, minZ = x, y, z
			maxX, maxY, maxZ = x, y, z
			first = false
			continue
		}

		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
		if z < minZ {
			minZ = z
		}
		if z > maxZ {
			maxZ = z
		}
	}

	return geo.Pos(minX, minY, minZ), geo.Pos(maxX, maxY, maxZ)
}
