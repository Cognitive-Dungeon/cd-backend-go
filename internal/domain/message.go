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
	Action  string          `json:"action"` // MOVE, WAIT, TALK
	Payload json.RawMessage `json:"payload"`
}

// Payloads
type MovePayload struct {
	Dx int `json:"dx"`
	Dy int `json:"dy"`
}

// ServerResponse - исходящее сообщение (Снапшот)
type ServerResponse struct {
	Type     string     `json:"type"` // INIT, UPDATE
	World    *GameWorld `json:"world,omitempty"`
	Player   *Entity    `json:"player,omitempty"`
	Entities []Entity   `json:"entities,omitempty"`
	Logs     []LogEntry `json:"logs,omitempty"`
}
