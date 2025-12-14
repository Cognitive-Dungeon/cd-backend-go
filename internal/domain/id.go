package domain

import (
	"fmt"
	"strconv"
)

// EntityID - упакованный идентификатор (Type + Level + Index)
type EntityID uint64

// Конфигурация битов
const (
	bitsIndex = 40
	bitsLevel = 16
	bitsType  = 8

	// Сдвиги
	shiftLevel = bitsIndex
	shiftType  = bitsIndex + bitsLevel

	// Маски (для извлечения значений)
	maskIndex = (1 << bitsIndex) - 1 // 0x000000FFFFFFFFFF
	maskLevel = (1 << bitsLevel) - 1 // 0xFFFF
	maskType  = (1 << bitsType) - 1  // 0xFF
)

// --- КОНСТРУКТОР ---

// PackEntityID создает ID из компонентов
func PackEntityID(typeID EntityType, levelID int16, index uint64) EntityID {
	id := uint64(index) & maskIndex
	id |= (uint64(levelID) & maskLevel) << shiftLevel
	id |= (uint64(typeID) & maskType) << shiftType
	return EntityID(id)
}

// --- МЕТОДЫ ДОСТУПА ---

func (id EntityID) Type() uint8 {
	return uint8((id >> shiftType) & maskType)
}

func (id EntityID) Level() int16 {
	return int16((id >> shiftLevel) & maskLevel)
}

func (id EntityID) Index() uint64 {
	return uint64(id & maskIndex)
}

// --- СЕРИАЛИЗАЦИЯ (Для фронтенда) ---

// MarshalJSON сериализует ID в строку, так как JS теряет точность для больших int64
func (id EntityID) MarshalJSON() ([]byte, error) {
	s := strconv.FormatUint(uint64(id), 10)
	return []byte(`"` + s + `"`), nil
}

// UnmarshalJSON парсит строку или число из JSON
func (id *EntityID) UnmarshalJSON(data []byte) error {
	// Удаляем кавычки, если есть
	if len(data) > 1 && data[0] == '"' && data[len(data)-1] == '"' {
		data = data[1 : len(data)-1]
	}
	val, err := strconv.ParseUint(string(data), 10, 64)
	if err != nil {
		return err
	}
	*id = EntityID(val)
	return nil
}

// String для логов: выводим красиво [Type:Lvl:Idx]
func (id EntityID) String() string {
	return fmt.Sprintf("[%d:%d:%d]", id.Type(), id.Level(), id.Index())
}
