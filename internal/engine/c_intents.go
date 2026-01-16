package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/pkg/ecs"
)

// IntentMove — валидированное намерение сдвинуться.
// Создается системой Input, потребляется системой Movement.
// Живет до конца фазы Logic.
type IntentMove struct {
	ecs.IntentMarker
	Dx, Dy int32
}

type IntentCast struct {
	ecs.IntentMarker
	SpellID  types.SpellID
	TargetID types.ObjectGuid
}
