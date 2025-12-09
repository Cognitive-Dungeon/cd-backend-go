package domain

import "strings"

// EventType - Внутренний числовой идентификатор события
type EventType uint8

// Event types constants
const (
	EventUnknown EventType = iota
	EventLevelTransition
	// Future events:
	// EventSpawnMonster = "SPAWN_MONSTER"
	// EventOpenDoor     = "OPEN_DOOR"
)

// Маппинг для конвертации JSON -> Domain
var eventStringToCmd = map[string]EventType{
	"LEVEL_TRANSITION": EventLevelTransition,
}

// Маппинг для логов Domain -> String
var eventCmdToString = map[EventType]string{
	EventLevelTransition: "LEVEL_TRANSITION",
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
