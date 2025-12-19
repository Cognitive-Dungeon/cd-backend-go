package enums

import "strings"

type EntityType uint8

const (
	EntityTypeUnknown EntityType = iota
	EntityTypePlayer
	EntityTypeNPC
	EntityTypeEnemy
	EntityTypeItem
	EntityTypeExit
	EntityTypeObject
)

var entityTypeToString = map[EntityType]string{
	EntityTypePlayer: "PLAYER",
	EntityTypeNPC:    "NPC",
	EntityTypeEnemy:  "ENEMY",
	EntityTypeItem:   "ITEM",
	EntityTypeExit:   "EXIT",
	EntityTypeObject: "OBJECT",
}

var entityTypeStringToType = map[string]EntityType{
	"PLAYER": EntityTypePlayer,
	"NPC":    EntityTypeNPC,
	"ENEMY":  EntityTypeEnemy,
	"EXIT":   EntityTypeExit,
	"OBJECT": EntityTypeObject,
}

// String возвращает строковое представление (для логов и дебага)
func (e EntityType) String() string {
	if val, ok := entityTypeToString[e]; ok {
		return val
	}
	return "UNKNOWN"
}

// ParseEntityType конвертирует строку в Enum (нужно для загрузки шаблонов/конфигов)
func ParseEntityType(s string) EntityType {
	upper := strings.ToUpper(s)
	if val, ok := entityTypeStringToType[upper]; ok {
		return val
	}
	return EntityTypeUnknown
}
