package worldmap

import (
	"cognitive-server/pkg/geo"
	"sync"
)

// TileInfo — расширенная информация о тайле для внешних систем
// (AI, Pathfinder, FOV, Gameplay).
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

// World — потокобезопасный контейнер бесконечного мира.
//
// Архитектура:
//   - Static Layer: Regions → Chunk (immutable)
//   - Dynamic Layer: Shards → SparseChunk (delta)
//
// Все runtime-изменения карты происходят ТОЛЬКО
// через SparseChunk.
type World struct {
	// --- Static Layer ---
	regionsMu sync.RWMutex
	Regions   map[geo.Location]*Region

	// --- Dynamic Layer ---
	Shards [ShardCount]*Shard

	// DefaultTile возвращается, если тайл отсутствует.
	DefaultTile Tile
}

// NewWorld создает новый пустой мир.
// С одним Region и шардами в количестве ShardCount
func NewWorld() *World {
	w := &World{
		Regions: make(map[geo.Location]*Region),
	}
	for i := 0; i < ShardCount; i++ {
		w.Shards[i] = &Shard{
			Deltas: make(map[geo.Location]*SparseChunk),
		}
	}
	return w
}

//
// -------------------- Coordinate helpers --------------------
//

// getRegionKey возвращает ключ региона по координатам чанка.
func getRegionKey(cx, cy int) geo.Location {
	return geo.Pos(cx>>RegionShift, cy>>RegionShift, 0)
}

// getShardIndex выбирает шард по координатам чанка.
func getShardIndex(chunkKey geo.Location) int {
	x, y, _ := chunkKey.XYZ()
	return (x ^ y) & ShardMask
}

//
// -------------------- Public API --------------------
//

// GetTile возвращает тайл по глобальным координатам.
//
// Алгоритм:
//  1. Проверка динамики (SparseChunk)
//  2. Чтение из статики (Chunk)
//  3. DefaultTile, если ничего нет
func (w *World) GetTile(pos geo.Location) Tile {
	chunkKey := GetChunkKey(pos)
	shard := w.Shards[getShardIndex(chunkKey)]
	lx, ly := GetLocalCoords(pos)

	// --- Dynamic Layer ---
	shard.mu.RLock()
	if delta := shard.GetDeltaUnsafe(chunkKey); delta != nil {
		if t, ok := delta.Get(lx, ly); ok {
			shard.mu.RUnlock()
			return t
		}
	}
	shard.mu.RUnlock()

	// --- Static Layer ---
	return w.getStaticTile(chunkKey, pos)
}

// IsSolidFast проверяет проходимость тайла максимально быстрым путём.
//
// Использует битовые маски:
//   - сначала динамика (Static + Delta)
//   - затем статика
func (w *World) IsSolidFast(pos geo.Location) bool {
	chunkKey := GetChunkKey(pos)
	shard := w.Shards[getShardIndex(chunkKey)]
	lx, ly := GetLocalCoords(pos)

	flatIdx := (ly << ChunkShift) | lx
	block := flatIdx >> 6
	bit := uint64(1) << (flatIdx & 63)

	// --- Dynamic Layer ---
	shard.mu.RLock()
	if delta := shard.GetDeltaUnsafe(chunkKey); delta != nil {
		res := (delta.SolidMask[block] & bit) != 0
		shard.mu.RUnlock()
		return res
	}
	shard.mu.RUnlock()

	// --- Static Layer ---
	base := w.getStaticChunk(chunkKey)
	if base == nil {
		return false
	}
	return base.IsSolidLocal(lx, ly)
}

// SetTile изменяет тайл по глобальным координатам.
//
// Все изменения пишутся ТОЛЬКО в SparseChunk.
func (w *World) SetTile(pos geo.Location, tile Tile) {
	chunkKey := GetChunkKey(pos)
	shard := w.Shards[getShardIndex(chunkKey)]

	shard.mu.Lock()
	defer shard.mu.Unlock()

	delta := shard.GetOrCreateDeltaUnsafe(chunkKey)

	// Lazy hydration масок
	if len(delta.Modifications) == 0 {
		base := w.getStaticChunk(chunkKey)
		delta.UpdateMasks(base)
	}

	lx, ly := GetLocalCoords(pos)
	delta.Set(lx, ly, tile)
}

// PutChunk загружает готовый статический чанк.
//
// Используется генераторами и загрузчиками карт.
func (w *World) PutChunk(chunkKey geo.Location, chunk *Chunk) {
	// База должна быть консистентной
	chunk.RebuildMasks()

	cx, cy, _ := chunkKey.XYZ()
	regionKey := getRegionKey(cx, cy)

	w.regionsMu.Lock()
	region, ok := w.Regions[regionKey]
	if !ok {
		region = &Region{}
		w.Regions[regionKey] = region
	}
	w.regionsMu.Unlock()

	rcX := cx & RegionMask
	rcY := cy & RegionMask
	region.PutChunk(rcX, rcY, chunk)
}

// PutChunkOnTilePos — legacy helper.
// Deprecated
func (w *World) PutChunkOnTilePos(tilePos geo.Location, c *Chunk) {
	w.PutChunk(GetChunkKey(tilePos), c)
}

// GetInfo возвращает расширенную информацию о тайле.
func (w *World) GetInfo(pos geo.Location) TileInfo {
	t := w.GetTile(pos)
	return TileInfo{
		Pos:      pos,
		Material: t.Material,
		Flags:    t.Flags,
	}
}

// Reset очищает весь мир (статика и динамика).
func (w *World) Reset() {
	// Static
	w.regionsMu.Lock()
	for k := range w.Regions {
		delete(w.Regions, k)
	}
	w.regionsMu.Unlock()

	// Dynamic
	for i := 0; i < ShardCount; i++ {
		s := w.Shards[i]
		s.mu.Lock()
		for k := range s.Deltas {
			delete(s.Deltas, k)
		}
		s.mu.Unlock()
	}
}

// GetChunk возвращает статический Chunk.
//
// ВАЖНО:
//   - Chunk immutable
//   - использовать ТОЛЬКО для чтения
func (w *World) GetChunk(chunkKey geo.Location) *Chunk {
	return w.getStaticChunk(chunkKey)
}

// RangeChunks итерируется по всем загруженным чанкам.
// Если f возвращает false, итерация прекращается.
// Deprecated
func (w *World) RangeChunks(f func(pos geo.Location, c *Chunk) bool) {
	w.regionsMu.RLock()
	defer w.regionsMu.RUnlock()

	for rPos, region := range w.Regions {
		rx, ry, rz := rPos.XYZ()
		for y := 0; y < 1<<RegionShift; y++ {
			for x := 0; x < 1<<RegionShift; x++ {
				c := region.GetChunk(x, y)
				if c == nil {
					continue
				}
				chunkPos := geo.Pos(
					(rx<<RegionShift)|x,
					(ry<<RegionShift)|y,
					rz,
				)
				if !f(chunkPos, c) {
					return
				}
			}
		}
	}
}

//
// -------------------- Internal helpers --------------------
//

func (w *World) getStaticChunk(chunkKey geo.Location) *Chunk {
	cx, cy, _ := chunkKey.XYZ()
	regionKey := getRegionKey(cx, cy)

	w.regionsMu.RLock()
	region, ok := w.Regions[regionKey]
	w.regionsMu.RUnlock()
	if !ok {
		return nil
	}

	return region.GetChunk(cx&RegionMask, cy&RegionMask)
}

func (w *World) getStaticTile(chunkKey geo.Location, pos geo.Location) Tile {
	chunk := w.getStaticChunk(chunkKey)
	if chunk == nil {
		return w.DefaultTile
	}
	lx, ly := GetLocalCoords(pos)
	t, _ := chunk.GetTile(lx, ly)
	return t
}
