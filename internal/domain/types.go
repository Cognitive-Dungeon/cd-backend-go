package domain

import (
	"strings"
)

type EntityType uint8

const (
	EntityTypeUnknown EntityType = iota // 0
	EntityTypePlayer                    // 1
	EntityTypeNPC                       // 2
	EntityTypeEnemy                     // 3
	EntityTypeItem                      // 4
	EntityTypeExit                      // 5
)

var entityTypeToString = map[EntityType]string{
	EntityTypePlayer: "PLAYER",
	EntityTypeNPC:    "NPC",
	EntityTypeEnemy:  "ENEMY",
	EntityTypeItem:   "ITEM",
	EntityTypeExit:   "EXIT",
}

var entityTypeStringToType = map[string]EntityType{
	"PLAYER": EntityTypePlayer,
	"NPC":    EntityTypeNPC,
	"ENEMY":  EntityTypeEnemy,
	"EXIT":   EntityTypeExit,
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

type ItemCategory uint8

const (
	ItemCategoryUnknown   ItemCategory = iota // 0
	ItemCategoryWeapon                        // 1
	ItemCategoryArmor                         // 2
	ItemCategoryPotion                        // 3
	ItemCategoryFood                          // 4
	ItemCategoryMisc                          // 5
	ItemCategoryContainer                     // 6
)

var itemCategoryToString = map[ItemCategory]string{
	ItemCategoryWeapon:    "WEAPON",
	ItemCategoryArmor:     "ARMOR",
	ItemCategoryPotion:    "POTION",
	ItemCategoryFood:      "FOOD",
	ItemCategoryMisc:      "MISC",
	ItemCategoryContainer: "CONTAINER",
}

var itemCategoryStringToType = map[string]ItemCategory{
	"WEAPON":    ItemCategoryWeapon,
	"ARMOR":     ItemCategoryArmor,
	"POTION":    ItemCategoryPotion,
	"FOOD":      ItemCategoryFood,
	"MISC":      ItemCategoryMisc,
	"CONTAINER": ItemCategoryContainer,
}

func (c ItemCategory) String() string {
	if val, ok := itemCategoryToString[c]; ok {
		return val
	}
	return "UNKNOWN"
}

func ParseItemCategory(s string) ItemCategory {
	upper := strings.ToUpper(s)
	if val, ok := itemCategoryStringToType[upper]; ok {
		return val
	}
	return ItemCategoryUnknown
}

type AIStateType uint8

const (
	StateUnknown AIStateType = iota
	AIStateIdle
	AIStateCombat
	AIStateFleeing
)

// ActionType - Внутренний числовой идентификатор действия
type ActionType uint8

// EventType - Внутренний числовой идентификатор события
type EventType uint8
