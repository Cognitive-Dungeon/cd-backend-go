package domain

// Типы сущностей
const (
	EntityTypePlayer = "PLAYER"
	EntityTypeNPC    = "NPC"
	EntityTypeEnemy  = "ENEMY"
	EntityTypeItem   = "ITEM"
	EntityTypeExit   = "EXIT"
)

// Стоимость действий в тиках (Time Units)
const (
	TimeCostMove        = 100
	TimeCostAttackLight = 80
	TimeCostAttackHeavy = 150
	TimeCostWait        = 50
	TimeCostInteract    = 50
	TimeCostPickup      = 50
	TimeCostDrop        = 30
	TimeCostUse         = 60
	TimeCostEquip       = 80
	TimeCostUnequip     = 60
)

// Параметры восприятия
const (
	VisionRadius = 8
	AggroRadius  = 10
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
