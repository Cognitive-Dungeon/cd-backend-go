package api

import "cognitive-server/internal/core/types"

// --- ВХОДЯЩИЕ (Client -> Server) ---

// MovePayload — параметры движения.
type MovePayload struct {
	Dx int `json:"dx"` // -1, 0, 1
	Dy int `json:"dy"` // -1, 0, 1
}

// CastPayload — параметры применения способности
type CastPayload struct {
	SpellID  types.SpellID    `json:"spellId"`
	TargetID types.ObjectGuid `json:"targetId"`
}

type ChatPayload struct {
	Message string `json:"message"`
}
