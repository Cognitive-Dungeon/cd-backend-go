package enums

import "cognitive-server/internal/core/types"

type EventType uint8

const (
	EventNone EventType = iota
	EventMoveRequest
	EventObjectMoved
	EventCastRequest
	EventDamageTaken
	EventDamageApply // Нанести урон (запрос к DamageSystem)
	EventHealApply   // Нанести лечение
	EventObjectDied  // Факт смерти (для DeathSystem, LootSystem, AI)
)

// MoveRequestEvent — намерение объекта сдвинуться на 1 тайл
type MoveRequestEvent struct {
	Object    types.ObjectGuid
	Direction Direction
}

// ObjectMovedEvent - Факт передвижения
type ObjectMovedEvent struct {
	Object types.ObjectGuid
	From   types.TilePos
	To     types.TilePos
}

type CastRequestEvent struct {
	Caster  types.ObjectGuid
	Target  types.ObjectGuid
	SpellID types.SpellID
}

type DamageTakenEvent struct {
	Target   types.ObjectGuid
	Attacker types.ObjectGuid
	Damage   int32
	IsFatal  bool
}

type DamageEvent struct {
	Source types.ObjectGuid
	Target types.ObjectGuid
	Amount int32
	School types.SpellSchool
}

// HealEvent
type HealEvent struct {
	Source types.ObjectGuid
	Target types.ObjectGuid
	Amount int32
}

type ObjectDiedEvent struct {
	Object types.ObjectGuid
	Killer types.ObjectGuid
}
