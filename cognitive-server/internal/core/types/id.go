package types

import (
	"fmt"
	"strconv"
)

// ObjectGuid — 64-битный идентификатор сущности.
//
// ObjectGuid является value-type и предназначен для дешёвого копирования,
// сериализации и сравнения.
//
// Формат битов (от старших к младшим):
//
//	[ Shard (8) | Type (8) | Generation (16) | Index (32) ]
//
// Где:
//   - Shard — идентификатор мира / сервера
//   - Type — тип сущности (Player, NPC, Item и т.д.)
//   - Generation — версия слота сущности (защита от устаревших ссылок)
//   - Index — индекс сущности в ECS-массиве
//
// Такой формат позволяет:
//   - быстро адресовать сущности в ECS
//   - определять принадлежность сущности миру
//   - безопасно обнаруживать stale references
type ObjectGuid uint64

// NilObjectGuid — нулевой идентификатор сущности.
//
// Используется как аналог nil для случаев, когда сущность отсутствует
// или ссылка ещё не инициализирована.
const NilObjectGuid ObjectGuid = 0

//
// ==========================
// Bit layout configuration
// ==========================
//

// Общее количество бит — 64.
const (
	// bitsIndex — количество бит под индекс сущности.
	// Позволяет адресовать до ~4.29 млрд сущностей в одном шарде.
	bitsIndex = 32

	// bitsGen — количество бит для поколения слота.
	bitsGen = 16

	// bitsType — количество бит для типа сущности.
	bitsType = 8

	// bitsShard — количество бит для идентификатора шарда.
	bitsShard = 8

	// Битовые сдвиги
	shiftGen   = bitsIndex
	shiftType  = bitsIndex + bitsGen
	shiftShard = bitsIndex + bitsGen + bitsType

	// Маски
	maskIndex = (1 << bitsIndex) - 1
	maskGen   = (1 << bitsGen) - 1
	maskType  = (1 << bitsType) - 1
	maskShard = (1 << bitsShard) - 1
)

// compile-time проверка корректности схемы битов
const _totalBits = bitsIndex + bitsGen + bitsType + bitsShard

// Если сумма битов != 64 — код не скомпилируется
type _objectGuidBitsCheck [64 - _totalBits]struct{}

//
// ==========================
// Constructors
// ==========================
//

// PackObjectGuid собирает ObjectGuid из составных частей.
//
// ! Fast-path функция:
//   - НЕ выполняет проверок диапазонов
//   - предполагает, что входные данные валидны
//
// Используется во внутренних hot-path участках (ECS, networking).
func PackObjectGuid(
	shard uint8,
	typ uint8,
	gen uint16,
	idx uint32,
) ObjectGuid {
	return ObjectGuid(
		(uint64(shard) << shiftShard) |
			(uint64(typ) << shiftType) |
			(uint64(gen) << shiftGen) |
			uint64(idx),
	)
}

// NewObjectGuid создаёт ObjectGuid с проверкой корректности значений.
//
// Рекомендуется использовать:
//   - на границах системы
//   - при сетевом вводе
//   - при десериализации
func NewObjectGuid(
	shard uint8,
	typ uint8,
	gen uint16,
	idx uint32,
) (ObjectGuid, error) {
	if idx > maskIndex {
		return 0, fmt.Errorf("objectguid: index overflow: %d", idx)
	}
	if gen > maskGen {
		return 0, fmt.Errorf("objectguid: generation overflow: %d", gen)
	}
	if typ > maskType {
		return 0, fmt.Errorf("objectguid: type overflow: %d", typ)
	}
	if shard > maskShard {
		return 0, fmt.Errorf("objectguid: shard overflow: %d", gen)
	}

	return PackObjectGuid(shard, typ, gen, idx), nil
}

//
// ==========================
// Accessors
// ==========================
//

// Index возвращает индекс сущности в ECS-массиве.
func (id ObjectGuid) Index() uint32 {
	return uint32(id & maskIndex)
}

// Generation возвращает поколение слота сущности.
func (id ObjectGuid) Generation() uint16 {
	return uint16((id >> shiftGen) & maskGen)
}

// Type возвращает тип сущности.
func (id ObjectGuid) Type() uint8 {
	return uint8((id >> shiftType) & maskType)
}

// Shard возвращает идентификатор шарда.
func (id ObjectGuid) Shard() uint8 {
	return uint8((id >> shiftShard) & maskShard)
}

//
// ==========================
// Utility methods
// ==========================
//

// IsNil проверяет, является ли идентификатор нулевым.
func (id ObjectGuid) IsNil() bool {
	return id == NilObjectGuid
}

// IsLocal проверяет, принадлежит ли сущность текущему шарду.
func (id ObjectGuid) IsLocal(currentShard uint8) bool {
	return id.Shard() == currentShard
}

// IsEqual сравнивает два идентификатора.
func (id ObjectGuid) IsEqual(other ObjectGuid) bool {
	return id == other
}

//
// ==========================
// Debug & formatting
// ==========================
//

// String возвращает человекочитаемое представление ObjectGuid.
//
// Предназначено для логирования и отладки.
func (id ObjectGuid) String() string {
	if id.IsNil() {
		return "<nil>"
	}

	return "ObjectGuid{" +
		"shard=" + strconv.Itoa(int(id.Shard())) +
		", type=" + strconv.Itoa(int(id.Type())) +
		", gen=" + strconv.Itoa(int(id.Generation())) +
		", idx=" + strconv.Itoa(int(id.Index())) +
		"}"
}

//
// ==========================
// JSON serialization
// ==========================
//

// MarshalJSON сериализует ObjectGuid в JSON как строку.
//
// Это предотвращает потерю точности при работе с JavaScript
// и другими средами без поддержки uint64.
func (id ObjectGuid) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(id), 10) + `"`), nil
}

// UnmarshalJSON десериализует ObjectGuid из JSON.
//
// Поддерживаются:
//   - строковое представление ("123456")
//   - числовое представление (123456)
func (id *ObjectGuid) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		*id = NilObjectGuid
		return nil
	}

	// строка
	if data[0] == '"' {
		// ""
		if len(data) == 2 {
			*id = NilObjectGuid
			return nil
		}

		v, err := strconv.ParseUint(string(data[1:len(data)-1]), 10, 64)
		if err != nil {
			return err
		}

		*id = ObjectGuid(v)
		return nil
	}

	// число
	v, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		return err
	}

	*id = ObjectGuid(v)
	return nil
}
