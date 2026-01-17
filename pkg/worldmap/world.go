package worldmap

import (
	"cognitive-server/pkg/geo"
	"sync"
)

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
