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
)

// Параметры восприятия
const (
	VisionRadius = 8
	AggroRadius  = 10
)
