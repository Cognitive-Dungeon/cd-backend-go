package api

import "encoding/json"

// --- ВХОДЯЩИЕ (Client -> Server) ---

// InboundMessage — конверт для любой команды от клиента.
type InboundMessage struct {
	Action  string          `json:"action"`
	Token   string          `json:"token,omitempty"` // <--- Добавили для LOGIN
	Payload json.RawMessage `json:"payload,omitempty"`
}
