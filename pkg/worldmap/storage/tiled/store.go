package tiled

import (
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/worldmap"
	"fmt"
)

// Store реализует worldmap.MapStorage для .tmj файлов.
type Store struct {
	chunks map[geo.Location]*worldmap.Chunk
}

// NewStore загружает карту, нарезает её на чанки и хранит в памяти.
func NewStore(path string, palette Palette, offset geo.Location) (*Store, error) {
	// 1. Читаем файл
	tmjMap, err := loadMapFile(path)
	if err != nil {
		return nil, err
	}

	s := &Store{
		chunks: make(map[geo.Location]*worldmap.Chunk),
	}

	// 2. Нарезаем
	if err := s.processMap(tmjMap, palette, offset); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Store) LoadChunk(pos geo.Location) (*worldmap.Chunk, error) {
	if c, ok := s.chunks[pos]; ok {
		return c, nil
	}
	return nil, fmt.Errorf("chunk not found: %s", pos)
}

func (s *Store) SaveChunk(pos geo.Location, c *worldmap.Chunk) error {
	return fmt.Errorf("tiled storage is read-only")
}

func (s *Store) processMap(m *Map, p Palette, offset geo.Location) error {
	ox, oy, oz := offset.XYZ()

	for _, layer := range m.Layers {
		// Обрабатываем только слои тайлов и чанки (infinite map)
		if !layer.Visible || layer.Type != "tilelayer" {
			continue
		}

		// Поддержка Infinite Maps (у Tiled бесконечные карты разбиты на чанки 16x16 или 32x32)
		if len(layer.Chunks) > 0 {
			for _, tChunk := range layer.Chunks {
				chunkGids, err := decodeGIDData(tChunk.Data, layer.Compression)
				if err != nil {
					return fmt.Errorf("failed to decode chunk data: %w", err)
				}

				// Передаем декодированные данные
				if err := s.processTileData(chunkGids, tChunk.Width, tChunk.X+ox, tChunk.Y+oy, oz, p); err != nil {
					return err
				}
			}
		} else {
			// Обычная карта (Fixed Size)
			data, err := layer.getTileData()
			if err != nil {
				return err
			}

			if err := s.processTileData(data, layer.Width, layer.X+ox, layer.Y+oy, oz, p); err != nil {
				return err
			}
		}
	}
	return nil
}

// processTileData проходит по массиву GID и заполняет чанки worldmap.
// data — плоский массив GID. width — ширина блока данных в тайлах.
// startX, startY — мировые координаты начала этого блока.
func (s *Store) processTileData(gids []uint32, width int, startX, startY, z int, p Palette) error {
	for i, gid := range gids {
		tile, ok := mapGid(gid, p)
		if !ok {
			continue // Пропускаем пустоту
		}

		// Координаты внутри блока данных
		lx := i % width
		ly := i / width

		// Глобальные координаты
		worldPos := geo.Pos(startX+lx, startY+ly, z)

		chunkKey := worldmap.GetChunkKey(worldPos)
		localX, localY := worldmap.GetLocalCoords(worldPos)

		chunk, exists := s.chunks[chunkKey]
		if !exists {
			chunk = worldmap.NewChunk()
			s.chunks[chunkKey] = chunk
		}

		chunk.SetTile(localX, localY, tile)
	}
	return nil
}
