package domain

import "encoding/json"

// LogEntry - запись в чате/логе
type LogEntry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Type      string `json:"type"` // INFO, COMBAT, SPEECH, ERROR
	Timestamp int64  `json:"timestamp"`
}

// ClientCommand - входящее сообщение от React
type ClientCommand struct {
	Token   string          `json:"token,omitempty"` // ID сущности, которая шлет команду
	Action  string          `json:"action"`          // MOVE, WAIT, TALK
	Payload json.RawMessage `json:"payload"`
}

// Payloads

// DirectionPayload: Для WASD движения или толчков
// Используется в: MOVE
type DirectionPayload struct {
	Dx int `json:"dx"`
	Dy int `json:"dy"`
}

// EntityPayload: Для взаимодействия с конкретным объектом
// Используется в: ATTACK, TALK, INSPECT, PICKUP, TRADE
type EntityPayload struct {
	TargetID string `json:"targetId"`
}

// PositionPayload: Для клика по карте или AoE магии (на будущее)
// Используется в: TELEPORT, CAST_AREA
type PositionPayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ServerResponse - исходящее сообщение (Снапшот)
type ServerResponse struct {
	Type           string     `json:"type"` // INIT, UPDATE, TURN
	World          *GameWorld `json:"world,omitempty"`
	Player         *Entity    `json:"player,omitempty"` // Для игрока-человека
	Entities       []Entity   `json:"entities,omitempty"`
	Logs           []LogEntry `json:"logs,omitempty"`
	ActiveEntityID string     `json:"activeEntityId,omitempty"`
}
