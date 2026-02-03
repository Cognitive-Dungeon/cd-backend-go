package worldmap

import (
	"cognitive-server/pkg/geo"
	"sync"
)

const (
	// RegionShift определяет размер региона в чанках как степень двойки.
	// 2^5 = 32 чанка по каждой оси (X, Y).
	RegionShift = 5

	// RegionMask используется для получения локальных координат чанка
	// внутри региона (rx, ry) через побитовое И.
	RegionMask = (1 << RegionShift) - 1

	// RegionArea — общее количество чанков в одном регионе.
	// 32 * 32 = 1024 чанка.
	RegionArea = (1 << RegionShift) * (1 << RegionShift)

	// ShardCount — количество шардов для динамических данных.
	//
	// Шарды используются для:
	//   - уменьшения lock contention
	//   - масштабирования записи/чтения дельт
	//
	// Каждая операция с динамикой попадает ровно в один Shard,
	// определяемый по хешу координат.
	ShardCount = 64

	// ShardMask используется для быстрого выбора шарда:
	// shardID = hash(location) & ShardMask
	ShardMask = ShardCount - 1
)

// Region — статический контейнер карты.
//
// Region хранит фиксированный массив из Chunk размером 32x32 (1024 чанка).
// Это:
//   - единица загрузки/выгрузки (streaming)
//   - единица кэширования статики
//
// ВАЖНО:
//   - Chunk внутри Region считаются неизменяемыми (immutable)
//   - любые изменения карты во время выполнения
//     должны происходить ТОЛЬКО через SparseChunk в Shard
//
// Region не содержит мьютексов:
//   - предполагается, что управление конкурентным доступом
//     происходит уровнем выше (World / RegionManager).
type Region struct {
	// Chunks — статический массив указателей на Chunk.
	//
	// Индексация:
	//   index = (ry << RegionShift) | rx
	//
	// Где:
	//   rx, ry ∈ [0..31] — локальные координаты чанка в регионе.
	Chunks [RegionArea]*Chunk
}

// GetChunk возвращает чанк по локальным координатам региона.
//
// Параметры:
//
//	rx, ry — координаты чанка внутри региона (0..31).
//
// Возвращает:
//
//	*Chunk — указатель на чанк или nil, если чанк не загружен.
//
// ВАЖНО:
//   - метод не делает проверок границ для скорости
//   - caller ОБЯЗАН гарантировать корректные rx, ry
func (r *Region) GetChunk(rx, ry int) *Chunk {
	return r.Chunks[(ry<<RegionShift)|rx]
}

// PutChunk помещает чанк в регион по локальным координатам.
//
// Используется:
//   - загрузчиком мира
//   - генератором карты
//
// ВАЖНО:
//   - после помещения в Region чанк считается immutable
//   - дальнейшие изменения должны идти через SparseChunk
func (r *Region) PutChunk(rx, ry int, c *Chunk) {
	r.Chunks[(ry<<RegionShift)|rx] = c
}

// Shard — динамический контейнер (Delta Layer).
//
// Shard хранит SparseChunk — изменения поверх статической карты.
//
// Назначение:
//   - изоляция конкурентных операций записи
//   - уменьшение блокировок при массовых изменениях мира
//
// Каждый Shard:
//   - защищён собственным RWMutex
//   - содержит независимую карту дельт
//
// Модель доступа:
//   - ЧТЕНИЕ: RLock → Get
//   - ЗАПИСЬ:  Lock → GetOrCreate → Set
type Shard struct {
	// mu защищает доступ к Deltas.
	// Используется для синхронизации чтения и записи динамики.
	mu sync.RWMutex

	// Deltas содержит SparseChunk для изменённых чанков.
	//
	// Ключ:
	//   geo.Location — глобальные координаты чанка
	//
	// Значение:
	//   *SparseChunk — дельта изменений для этого чанка
	Deltas map[geo.Location]*SparseChunk
}

// GetDeltaUnsafe возвращает SparseChunk без синхронизации.
//
// ВНИМАНИЕ:
//   - метод НЕ потокобезопасен
//   - caller ОБЯЗАН удерживать Lock или RLock шарда
//
// Используется в горячих путях (pathfinding, FOV),
// где блокировки берутся уровнем выше.
func (s *Shard) GetDeltaUnsafe(key geo.Location) *SparseChunk {
	return s.Deltas[key]
}

// GetOrCreateDeltaUnsafe возвращает существующий SparseChunk
// или создаёт новый, если он отсутствует.
//
// ВНИМАНИЕ:
//   - метод НЕ потокобезопасен
//   - caller ОБЯЗАН удерживать Lock шарда
//
// Используется при записи изменений в мир.
func (s *Shard) GetOrCreateDeltaUnsafe(key geo.Location) *SparseChunk {
	d, ok := s.Deltas[key]
	if !ok {
		d = NewSparseChunk()
		s.Deltas[key] = d
	}
	return d
}
