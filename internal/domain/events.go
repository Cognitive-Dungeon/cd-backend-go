package domain

import "strings"

// Event types constants
const (
	EventUnknown EventType = iota
	EventLevelTransition
	EventItemReaction // Для событий живых предметов (будет обрабатываться микросервисом)
	// Future events:
	// EventSpawnMonster = "SPAWN_MONSTER"
	// EventOpenDoor     = "OPEN_DOOR"
)

// Маппинг для конвертации JSON -> Domain
var eventStringToCmd = map[string]EventType{
	"LEVEL_TRANSITION": EventLevelTransition,
	"ITEM_EVENT":       EventItemReaction,
}

// Маппинг для логов Domain -> String
var eventCmdToString = map[EventType]string{
	EventLevelTransition: "LEVEL_TRANSITION",
	EventItemReaction:    "ITEM_EVENT",
}

// ParseEvent конвертирует строку из JSON в EventType
func ParseEvent(s string) EventType {
	// Делаем нечувствительным к регистру для надежности
	upper := strings.ToUpper(s)
	if val, ok := eventStringToCmd[upper]; ok {
		return val
	}
	return EventUnknown
}

// String реализует интерфейс Stringer (для fmt.Printf)
func (a EventType) String() string {
	if val, ok := eventCmdToString[a]; ok {
		return val
	}
	return "UNKNOWN"
}

// AttackRequested Запрос события нанесения атаки
type AttackRequested struct {
	Attacker *Entity
	Target   *Entity
	World    *GameWorld
}

// DamageInflicted Урон был рассчитан и нанесен.
type DamageInflicted struct {
	Target   *Entity
	Attacker *Entity
	Amount   int
	IsFatal  bool
	World    *GameWorld
}

// EntityDied Сущность погибла.
type EntityDied struct {
	Entity *Entity
	Killer *Entity
	World  *GameWorld
}

// LogMessage Нужно записать что-то в лог инстанса.
type LogMessage struct {
	World *GameWorld
	Text  string
	Type  string // "INFO", "COMBAT", "ERROR"
}

type MoveRequested struct {
	Actor *Entity
	Position
	World *GameWorld
}

type EntityMoved struct {
	Actor        *Entity
	FromPosition Position
	ToPosition   Position
	World        *GameWorld
}

// --- Inventory ---
type InventoryActionPayload struct { // Общая структура для простых действий
	Actor  *Entity
	ItemID EntityID
	World  *GameWorld
	Count  int // Для Drop (опционально)
}

// --- Interaction ---
type InteractRequested struct {
	Actor    *Entity
	TargetID EntityID
	World    *GameWorld
}

// Событие для смены уровня (результат лестницы)
type LevelTransition struct {
	Actor       *Entity
	TargetLevel int
	TargetPos   Position
}
