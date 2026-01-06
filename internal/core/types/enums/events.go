package enums

type EventType uint8

const (
	EventTypeUnknown EventType = iota

	// Combat
	EventTypeAttackRequested
	EventTypeDamageInflicted
	EventTypeEntityDied

	// Movement
	EventTypeMoveRequested
	EventTypeEntityMoved // Для триггеров ловушек или обновления тумана войны

	// Inventory
	EventTypePickupRequested
	EventTypeDropRequested
	EventTypeEquipRequested
	EventTypeUnequipRequested
	EventTypeUseRequested
	EventTypeItemUsed // Факт использования (для логов/квестов)

	// Interaction
	EventTypeInteractRequested
	EventTypeLevelTransition // Для лестниц

	// System
	EventTypeLogMessage

	EventTypeCount
)
