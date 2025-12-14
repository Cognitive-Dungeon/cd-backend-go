package domain

// Типы сущностей
const (
	EntityTypePlayer = "PLAYER"
	EntityTypeNPC    = "NPC"
	EntityTypeEnemy  = "ENEMY"
	EntityTypeItem   = "ITEM"
	EntityTypeExit   = "EXIT"
)

// Категории предметов
const (
	ItemCategoryWeapon    = "weapon"
	ItemCategoryArmor     = "armor"
	ItemCategoryPotion    = "potion"
	ItemCategoryFood      = "food"
	ItemCategoryMisc      = "misc"
	ItemCategoryContainer = "container"
)

// ActionType - Внутренний числовой идентификатор действия
type ActionType uint8

// EventType - Внутренний числовой идентификатор события
type EventType uint8
