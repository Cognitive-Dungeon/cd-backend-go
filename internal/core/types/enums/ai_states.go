package enums

type AIStateType uint8

const (
	StateUnknown AIStateType = iota
	AIStateIdle
	AIStateCombat
	AIStateFleeing
)
