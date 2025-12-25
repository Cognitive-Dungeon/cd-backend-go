package enums

type EventType uint8

const (
	EventTypeUnknown EventType = iota

	// Combat
	EventTypeAttackRequested
	EventTypeDamageInflicted
	EventTypeEntityDied

	// World
	EventTypeEntityMoved
	EventTypeItemPickedUp

	// System
	EventTypeLogMessage

	EventTypeCount
)
