package api

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
	Spells         []SpellView  `json:"spells,omitempty"`
}

// GridMeta — размеры мира.
type GridMeta struct {
	Width  int `json:"w"`
	Height int `json:"h"`
}

type ChatMessage struct {
	Type       uint8  `json:"type"`
	SenderName string `json:"senderName"`
	SenderGuid string `json:"senderGuid"`
	Text       string `json:"text"`
}

type AsyncMessage struct {
	Type string      `json:"type"` // "CHAT"
	Data interface{} `json:"data"`
}
