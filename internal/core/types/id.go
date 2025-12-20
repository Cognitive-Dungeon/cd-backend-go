package types

import (
	"fmt"
	"strconv"
)

// EntityID — 64-битный идентификатор сущности.
//
// EntityID является value-type и предназначен для дешёвого копирования,
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
type EntityID uint64

// NilEntityID — нулевой идентификатор сущности.
//
// Используется как аналог nil для случаев, когда сущность отсутствует
// или ссылка ещё не инициализирована.
const NilEntityID EntityID = 0

// Конфигурация битов EntityID.
//
// Общее количество бит — 64.
const (
	// bitsIndex — количество бит, выделенных под индекс сущности.
	// Позволяет адресовать до ~4.29 миллиарда сущностей в рамках одного шарда.
	bitsIndex = 32

	// bitsGen — количество бит для поколения слота.
	// Используется для защиты от использования устаревших ссылок.
	bitsGen = 16

	// bitsType — количество бит для типа сущности.
	// Позволяет определить до 256 различных типов сущностей.
	bitsType = 8

	// bitsShard — количество бит для идентификатора шарда (мира).
	// Позволяет использовать до 256 миров / серверов.
	bitsShard = 8

	// Сдвиги битов
	shiftGen   = bitsIndex
	shiftType  = bitsIndex + bitsGen
	shiftShard = bitsIndex + bitsGen + bitsType

	// Маски для извлечения значений
	maskIndex = (1 << bitsIndex) - 1
	maskGen   = (1 << bitsGen) - 1
	maskType  = (1 << bitsType) - 1
	maskShard = (1 << bitsShard) - 1
)

// PackEntityID собирает EntityID из составных частей.
//
// Параметры:
//   - shardID — идентификатор текущего мира / сервера
//   - typeID — тип сущности
//   - gen — поколение слота сущности
//   - index — индекс сущности в ECS-массиве
//
// Функция не выполняет проверок диапазонов значений и предполагает,
// что входные данные валидны.
func PackEntityID(
	shardID uint8,
	typeID uint8,
	gen uint16,
	index uint32,
) EntityID {
	return EntityID(
		(uint64(shardID) << shiftShard) |
			(uint64(typeID) << shiftType) |
			(uint64(gen) << shiftGen) |
			uint64(index),
	)
}

// Index возвращает индекс сущности в ECS-массиве.
func (id EntityID) Index() uint32 {
	return uint32(id & maskIndex)
}

// Generation возвращает поколение слота сущности.
//
// Используется для обнаружения устаревших ссылок на уничтоженные сущности.
func (id EntityID) Generation() uint16 {
	return uint16((id >> shiftGen) & maskGen)
}

// Type возвращает тип сущности.
func (id EntityID) Type() uint8 {
	return uint8((id >> shiftType) & maskType)
}

// Shard возвращает идентификатор шарда, которому принадлежит сущность.
func (id EntityID) Shard() uint8 {
	return uint8((id >> shiftShard) & maskShard)
}

// IsNil проверяет, является ли идентификатор нулевым.
func (id EntityID) IsNil() bool {
	return id == NilEntityID
}

// IsLocal проверяет, принадлежит ли сущность текущему шарду.
func (id EntityID) IsLocal(currentShard uint8) bool {
	return id.Shard() == currentShard
}

// String возвращает человекочитаемое строковое представление EntityID.
//
// Предназначено для логирования и отладки.
func (id EntityID) String() string {
	if id.IsNil() {
		return "<nil>"
	}

	return fmt.Sprintf(
		"[shard=%d type=%d gen=%d idx=%d]",
		id.Shard(),
		// TODO: Взять код из types/enums/entities
		id.Type(),
		id.Generation(),
		id.Index(),
	)
}

// MarshalJSON сериализует EntityID в JSON как строку.
//
// Это необходимо для предотвращения потери точности при работе с
// JavaScript и другими средами, не поддерживающими uint64.
func (id EntityID) MarshalJSON() ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(uint64(id), 10) + `"`), nil
}

// UnmarshalJSON десериализует EntityID из JSON.
//
// Поддерживаются как строковое, так и числовое представление.
func (id *EntityID) UnmarshalJSON(data []byte) error {
	s := string(data)

	if len(s) > 1 && s[0] == '"' {
		s = s[1 : len(s)-1]
	}

	if s == "" {
		*id = NilEntityID
		return nil
	}

	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return err
	}

	*id = EntityID(v)
	return nil
}
