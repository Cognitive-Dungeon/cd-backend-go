package domain

import "encoding/json"

// InternalCommand - оптимизированная команда для движка.
// Использует ActionType вместо string.
type InternalCommand struct {
	Action  ActionType      // Число! Быстро и безопасно.
	Token   EntityID        // ID сущности (Actor)
	Payload json.RawMessage // Сырые данные (парсятся хендлером)
}
