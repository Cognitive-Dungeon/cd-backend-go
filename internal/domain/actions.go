package domain

import "strings"

// ActionType - Внутренний числовой идентификатор действия
type ActionType uint8

const (
	ActionUnknown ActionType = iota
	ActionInit
	ActionMove
	ActionAttack
	ActionWait
	ActionTalk
	ActionInteract
	// В будущем: ActionTrade, ActionUseItem...
)

// Маппинг для конвертации JSON -> Domain
var actionStringToCmd = map[string]ActionType{
	"INIT":     ActionInit,
	"MOVE":     ActionMove,
	"ATTACK":   ActionAttack,
	"WAIT":     ActionWait,
	"TALK":     ActionTalk,
	"INTERACT": ActionInteract,
}

// Маппинг для логов Domain -> String
var actionCmdToString = map[ActionType]string{
	ActionInit:     "INIT",
	ActionMove:     "MOVE",
	ActionAttack:   "ATTACK",
	ActionWait:     "WAIT",
	ActionTalk:     "TALK",
	ActionInteract: "INTERACT",
}

// ParseAction конвертирует строку из JSON в ActionType
func ParseAction(s string) ActionType {
	// Делаем нечувствительным к регистру для надежности
	upper := strings.ToUpper(s)
	if val, ok := actionStringToCmd[upper]; ok {
		return val
	}
	return ActionUnknown
}

// String реализует интерфейс Stringer (для fmt.Printf)
func (a ActionType) String() string {
	if val, ok := actionCmdToString[a]; ok {
		return val
	}
	return "UNKNOWN"
}
