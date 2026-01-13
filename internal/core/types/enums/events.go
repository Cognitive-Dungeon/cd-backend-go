package enums

import "cognitive-server/internal/core/types"

type EventType uint8

const (
	EventNone EventType = iota
	EventMoveRequest
	EventObjectMoved
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
