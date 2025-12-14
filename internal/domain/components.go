package domain

import "encoding/json"

// --- КОМПОНЕНТЫ ---

// RenderComponent - Визуализация (Клиент)
type RenderComponent struct {
	Symbol byte   `json:"symbol"` // Символ отображения (g-гоблин, $-монетка)
	Color  string `json:"color"`
}

// StatsComponent - Характеристики и Ресурсы
type StatsComponent struct {
	HP         int  `json:"hp"`
	MaxHP      int  `json:"maxHp"`
	Stamina    int  `json:"stamina"`
	MaxStamina int  `json:"maxStamina"`
	Strength   int  `json:"strength"`
	Gold       int  `json:"gold"`
	IsDead     bool `json:"isDead"`
}

// AIComponent - Мозги, Поведение и Время
// Примечание: У игрока тоже есть этот компонент, чтобы хранить NextActionTick
type AIComponent struct {
	IsHostile      bool        `json:"isHostile"`
	State          AIStateType `json:"state,omitempty"`       // "IDLE"
	NextActionTick int         `json:"nextActionTick"`        // <-- Очередь ходов
	Personality    string      `json:"personality,omitempty"` // "Cowardly"
}

// NarrativeComponent - Данные для LLM и Осмотра
type NarrativeComponent struct {
	Description string `json:"description"` // "Грязный гоблин с ржавым ножом"
}

// VisionComponent - настройки зрения
type VisionComponent struct {
	Radius     int  `json:"radius"`
	Omniscient bool `json:"omniscient"` // Всеведенье

	// Caching Optimization
	CachedVisibleTiles map[int]bool `json:"-"` // Не сериализуем кэш
	IsDirty            bool         `json:"-"` // Флаг для пересчета
}

// MemoryComponent - туман войны
type MemoryComponent struct {
	// Храним набор исследованных ID для каждого уровня отдельно.
	ExploredPerLevel map[int]map[int]bool `json:"exploredPerLevel"`
}

// TriggerComponent описывает, что происходит при взаимодействии с сущностью.
type TriggerComponent struct {
	// OnInteract содержит JSON-объект события, которое сработает при команде INTERACT.
	// Например: {"event": "LEVEL_TRANSITION", "targetLevel": 1, "targetPosId": "exit_up"}
	OnInteract json.RawMessage `json:"onInteract,omitempty"`
}
