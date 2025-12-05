package domain

type Stats struct {
	HP         int `json:"hp"`
	MaxHP      int `json:"maxHp"`
	Stamina    int `json:"stamina"`
	MaxStamina int `json:"maxStamina"`
	Gold       int `json:"gold"`
	Strength   int `json:"strength"`
}

type Entity struct {
	// Core
	ID     string `json:"id"`
	Type   string `json:"type"`   // см. constants.go
	Label  string `json:"label"`  // Визуальная метка
	Symbol string `json:"symbol"` // ASCII символ
	Color  string `json:"color"`  // CSS класс или hex

	// Physics
	Pos Position `json:"pos"`

	// Data
	Stats Stats  `json:"stats"`
	Name  string `json:"name"`

	// Flags
	IsHostile bool `json:"isHostile"`
	IsDead    bool `json:"isDead"`

	// Time System
	NextActionTick int `json:"nextActionTick"`

	// AI & Narrative
	Personality string `json:"personality,omitempty"`
	AIState     string `json:"aiState,omitempty"` // IDLE, COMBAT, FLEEING
}
