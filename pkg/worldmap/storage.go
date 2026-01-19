package worldmap

import "cognitive-server/pkg/geo"

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

// LoadRegion загружает квадратную область чанков (например, вокруг игрока).
func (l *Loader) LoadRegion(center geo.Location, radiusChunks int) error {
	// Превращаем центр в координаты чанка
	cx, cy, cz := GetChunkKey(center).XYZ()

	// Простая итерация по квадрату
	for dy := -radiusChunks; dy <= radiusChunks; dy++ {
		for dx := -radiusChunks; dx <= radiusChunks; dx++ {
			// Формируем ключ чанка
			targetKey := geo.Pos(cx+dx, cy+dy, cz)

			// Пропускаем, если уже есть (опционально)
			if l.world.GetChunk(targetKey) != nil {
				continue
			}

			// Грузим из хранилища
			chunk, err := l.storage.LoadChunk(targetKey)
			if err != nil {
				// TODO: Обработать ошибки загрузки чатков
				// Тут стратегия обработки ошибок:
				// Можно вернуть ошибку, можно создать пустой чанк, можно логировать.
				// Пока просто пропустим (будет "DefaultTile").
				continue
			}

			// Вставляем в мир (Batch Operation)
			l.world.PutChunk(targetKey, chunk)
		}
	}
	return nil
}
