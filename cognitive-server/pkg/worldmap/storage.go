package worldmap

import (
	"cognitive-server/pkg/geo"
)

// MapStorage — абстракция источника данных карты.
type MapStorage interface {
	// LoadChunk загружает один чанк по его координатам.
	LoadChunk(pos geo.Location) (*Chunk, error)

	// SaveChunk сохраняет состояние чанка.
	SaveChunk(pos geo.Location, c *Chunk) error
}

// Loader — вспомогательная структура для массовой загрузки.
type Loader struct {
	world   *World
	storage MapStorage
}

func NewLoader(w *World, s MapStorage) *Loader {
	return &Loader{world: w, storage: s}
}

// LoadRegionChunkCenter загружает квадратную область чанков (например, вокруг игрока).
func (l *Loader) LoadRegionChunkCenter(centerChunk geo.Location, radius int) (loaded int, err error) {
	cx, cy, cz := centerChunk.XYZ()

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			key := geo.Pos(cx+dx, cy+dy, cz)

			// Уже загружен
			if l.world.GetChunk(key) != nil {
				continue
			}

			chunk, e := l.storage.LoadChunk(key)
			if e != nil {
				continue // норм: outside map
			}

			l.world.PutChunk(key, chunk)
			loaded++
		}
	}

	return loaded, nil
}

func (l *Loader) LoadRegionTileCenter(tileCenter geo.Location, radius int) (int, error) {
	return l.LoadRegionChunkCenter(GetChunkKey(tileCenter), radius)
}
