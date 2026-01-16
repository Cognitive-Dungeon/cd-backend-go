package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/ecs"
)

// CmdMove — сырой запрос от клиента/сети.
// Живет до конца фазы Input.
type CmdMove struct {
	ecs.RequestMarker
	Direction enums.Direction
}

type CmdCast struct {
	ecs.RequestMarker
	SpellID  types.SpellID
	TargetID types.ObjectGuid
}
