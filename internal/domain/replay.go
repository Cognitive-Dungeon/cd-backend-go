package domain

import "encoding/json"

// ReplayAction - это запись одного действия извне (от игрока)
type ReplayAction struct {
	Tick    int             `json:"tick"`
	Token   EntityID        `json:"token"`   // Кто сделал
	Action  ActionType      `json:"action"`  // Что сделал
	Payload json.RawMessage `json:"payload"` // С какими параметрами
}

// ReplaySession - полная запись партии
type ReplaySession struct {
	LevelID     int             `json:"levelId"`
	Seed        int64           `json:"seed"` // Зерно генерации мира и рандома
	Timestamp   int64           `json:"timestamp"`
	PlayerState json.RawMessage `json:"playerState,omitempty"`
	Actions     []ReplayAction  `json:"actions"`
}
