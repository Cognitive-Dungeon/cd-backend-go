package domain

// Стоимость действий в тиках (Time Units)
const (
	TimeCostMove        = 100
	TimeCostAttackLight = 80
	TimeCostAttackHeavy = 150
	TimeCostWait        = 50
	TimeCostInteract    = 50
	TimeCostPickup      = 50
	TimeCostDrop        = 30
	TimeCostUse         = 60
	TimeCostEquip       = 80
	TimeCostUnequip     = 60
)

// Параметры восприятия
const (
	VisionRadius = 8
	AggroRadius  = 10
)

var (
	DirNorth = Position{X: 0, Y: -1}
	DirSouth = Position{X: 0, Y: 1}
	DirWest  = Position{X: -1, Y: 0}
	DirEast  = Position{X: 1, Y: 0}

	DirNorthWest = Position{X: -1, Y: -1}
	DirNorthEast = Position{X: 1, Y: -1}
	DirSouthWest = Position{X: -1, Y: 1}
	DirSouthEast = Position{X: 1, Y: 1}
)
