package api

import (
	"encoding/json"
)

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

// --- DTO для Визуализации (View Layer) ---

// TileView - это то, как клиент видит клетку.
type TileView struct {
	X int `json:"x"`
	Y int `json:"y"`

	// Визуал
	Symbol string `json:"symbol"`
	Color  string `json:"color"`

	// Флаги для рендера
	IsWall     bool `json:"isWall"`
	IsVisible  bool `json:"isVisible"`
	IsExplored bool `json:"isExplored"`
}

// GridMeta - Общие размеры карты которые клиент должен быть готов отрисовать
type GridMeta struct {
	Width  int `json:"w"`
	Height int `json:"h"`
}

type StatsView struct {
	HP         int  `json:"hp"`
	MaxHP      int  `json:"maxHp"`
	Stamina    int  `json:"stamina,omitempty"` // Врагу знать не обязательно
	MaxStamina int  `json:"maxStamina,omitempty"`
	Gold       int  `json:"gold,omitempty"` // Врагу знать не обязательно
	Strength   int  `json:"strength,omitempty"`
	IsDead     bool `json:"isDead"`
}

// EntityView - Единое представление Entity
type EntityView struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`

	Pos struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"pos"`

	Render struct {
		Symbol string `json:"symbol"`
		Color  string `json:"color"`
	} `json:"render"`

	// Stats присутствует, только если это "Я" или если есть право видеть статы (например, труп)
	Stats *StatsView `json:"stats,omitempty"`
}

// ServerResponse - Снапшот состояния для конкретного клиента
type ServerResponse struct {
	Type           string       `json:"type"`
	Tick           int          `json:"tick"`
	ActiveEntityID string       `json:"activeEntityId,omitempty"`
	MyEntityID     string       `json:"myEntityId,omitempty"`
	Grid           *GridMeta    `json:"grid,omitempty"`
	Map            []TileView   `json:"map,omitempty"`
	Entities       []EntityView `json:"entities,omitempty"`
	Logs           []LogEntry   `json:"logs,omitempty"`
}
