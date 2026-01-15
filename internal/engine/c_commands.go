package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
)

// CmdMove — сырой запрос от клиента/сети.
// Живет до конца фазы Input.
type CmdMove struct {
	Direction enums.Direction
}

type CmdCast struct {
	SpellID  types.SpellID
	TargetID types.ObjectGuid
}
