package engine

import "cognitive-server/internal/core/types"

// IntentMove — валидированное намерение сдвинуться.
// Создается системой Input, потребляется системой Movement.
// Живет до конца фазы Logic.
type IntentMove struct {
	Dx, Dy int32
}

type IntentCast struct {
	SpellID  types.SpellID
	TargetID types.ObjectGuid
}
