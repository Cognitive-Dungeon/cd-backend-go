package types

import (
	"errors"
	"hash/fnv"
	"strconv"
	"sync"
	"time"
)

// EntityID - это 64-битный уникальный идентификатор.
// Структура: [ Timestamp (41) | ShardID (8) | TypeID (5) | Sequence (10) ]
type EntityID uint64

const (
	// Конфигурация битов
	bitsSeq   = 10
	bitsType  = 5
	bitsShard = 8
	bitsTime  = 41

	// Сдвиги
	shiftType  = bitsSeq
	shiftShard = bitsSeq + bitsType
	shiftTime  = bitsSeq + bitsType + bitsShard

	// Маски
	maskSeq   = (1 << bitsSeq) - 1
	maskType  = (1 << bitsType) - 1
	maskShard = (1 << bitsShard) - 1

	// Эпоха: 4 Декабря 2025 года
	customEpoch = 1764806400000
)

// NilID - нулевой идентификатор (аналог nil)
const NilID EntityID = 0

// --- ГЕНЕРАТОР ---

type IDGenerator struct {
	mu       sync.Mutex
	shardID  uint64
	lastTime uint64
	sequence uint64
}

// NewGenerator создает генератор для конкретного шарда (сервера)
func NewGenerator(shardID uint8) *IDGenerator {
	return &IDGenerator{
		shardID: uint64(shardID),
	}
}

// NextID создает новый уникальный ID указанного типа
func (g *IDGenerator) NextID(typeID uint8) (EntityID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := uint64(time.Now().UnixMilli()) - customEpoch

	if now < g.lastTime {
		return 0, errors.New("clock moved backwards")
	}

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & maskSeq
		if g.sequence == 0 {
			// Ждем следующую миллисекунду при переполнении
			for now <= g.lastTime {
				now = uint64(time.Now().UnixMilli()) - customEpoch
			}
		}
	} else {
		g.sequence = 0
		g.lastTime = now
	}

	id := (now << shiftTime) |
		(g.shardID << shiftShard) |
		(uint64(typeID&uint8(maskType)) << shiftType) |
		(g.sequence)

	return EntityID(id), nil
}

// --- МЕТОДЫ EntityID ---

// Sequence возвращает порядковый номер ID в пределах миллисекунды.
// Возвращает значение в диапазоне 0-1023.
func (id EntityID) Sequence() uint16 {
	return uint16(id & maskSeq)
}

func (id EntityID) Type() uint8 {
	return uint8((id >> shiftType) & maskType)
}

func (id EntityID) Shard() uint8 {
	return uint8((id >> shiftShard) & maskShard)
}

func (id EntityID) Time() time.Time {
	ts := (uint64(id) >> shiftTime) + customEpoch
	return time.UnixMilli(int64(ts))
}

func (id EntityID) IsNil() bool {
	return id == 0
}

// String возвращает строковое представление (для логов)
func (id EntityID) String() string {
	return strconv.FormatUint(uint64(id), 10)
}

// --- JSON INTERFACE ---

// MarshalJSON превращает uint64 в string для JSON (JS safe)
func (id EntityID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + id.String() + `"`), nil
}

// UnmarshalJSON парсит строку или число из JSON
func (id *EntityID) UnmarshalJSON(data []byte) error {
	s := string(data)
	// Убираем кавычки, если есть
	if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	// Обработка пустой строки
	if s == "" {
		*id = EntityID(0)
		return nil
	}
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}
	*id = EntityID(val)
	return nil
}

// GenerateDeterministicID генерирует ID на основе строки.
// Это нужно для статических объектов (выходы, ключевые NPC), ID которых
// должны быть известны до их создания.
func GenerateDeterministicID(seed string, typeID uint8) EntityID {
	// 1. Хешируем строку в uint64
	h := fnv.New64a()
	h.Write([]byte(seed))
	hash := h.Sum64()

	// 2. Накладываем маску, чтобы сохранить структуру нашего Snowflake ID
	// Очищаем биты типа
	hash &^= uint64(maskType) << shiftType
	// Устанавливаем биты типа
	hash |= uint64(typeID&maskType) << shiftType

	return EntityID(hash)
}

// GenerateRandomID генерирует случайный ID,
// используя внешний RNG, чтобы сохранить детерминизм генерации уровня.
func GenerateRandomID(rngUint64 uint64, typeID uint8) EntityID {
	id := rngUint64
	id &^= (uint64(maskType) << shiftType)
	id |= (uint64(typeID&maskType) << shiftType)
	return EntityID(id)
}
