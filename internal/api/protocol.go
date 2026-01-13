package api

import (
	"encoding/json"
)

// --- ВХОДЯЩИЕ (Client -> Server) ---

// InboundMessage — конверт для любой команды от клиента.
type InboundMessage struct {
	Action  string          `json:"action"`
	Token   string          `json:"token,omitempty"` // <--- Добавили для LOGIN
	Payload json.RawMessage `json:"payload,omitempty"`
}

// MovePayload — параметры движения.
type MovePayload struct {
	Dx int `json:"dx"` // -1, 0, 1
	Dy int `json:"dy"` // -1, 0, 1
}

// --- ИСХОДЯЩИЕ (Server -> Client) ---

// ServerResponse — главный пакет обновления мира.
// Полностью соответствует ожиданиям debug_client.html.
type ServerResponse struct {
	Type           string       `json:"type"`           // Всегда "UPDATE"
	Tick           int64        `json:"tick"`           // Номер тика
	MyEntityID     string       `json:"myEntityId"`     // ID игрока
	ActiveEntityID string       `json:"activeEntityId"` // ID того, чей сейчас ход (для UI)
	Grid           *GridMeta    `json:"grid"`           // Размеры карты
	Map            []TileView   `json:"map"`            // Тайлы
	Entities       []EntityView `json:"entities"`       // Сущности
}

// GridMeta — размеры мира.
type GridMeta struct {
	Width  int `json:"w"`
	Height int `json:"h"`
}

// TileView — описание одной клетки карты.
type TileView struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Symbol    string `json:"symbol"`
	Color     string `json:"color"`
	IsWall    bool   `json:"isWall"`
	IsVisible bool   `json:"isVisible"` // Для тумана войны
}

// EntityView — описание объекта для отрисовки.
type EntityView struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "UNIT", "ITEM"
	Name string `json:"name"`

	Pos struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"pos"`

	Render struct {
		Symbol string `json:"symbol"`
		Color  string `json:"color"`
	} `json:"render"`

	Stats *StatsView `json:"stats,omitempty"`
}

// StatsView — характеристики (HP/Mana).
type StatsView struct {
	HP    int `json:"hp"`
	MaxHP int `json:"maxHp"`
}
