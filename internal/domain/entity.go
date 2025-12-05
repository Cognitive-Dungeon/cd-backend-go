package domain

// --- КОМПОНЕНТЫ ---

// RenderComponent - Визуализация (Клиент)
type RenderComponent struct {
	Symbol string `json:"symbol"` // Символ отображения (g-гоблин, $-монетка)
	Color  string `json:"color"`
	Label  string `json:"label"` // Метка для таргетинга (A, B, @)
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
	IsHostile      bool   `json:"isHostile"`
	State          string `json:"state,omitempty"`       // "IDLE"
	NextActionTick int    `json:"nextActionTick"`        // <-- Очередь ходов
	Personality    string `json:"personality,omitempty"` // "Cowardly"
}

// NarrativeComponent - Данные для LLM и Осмотра
type NarrativeComponent struct {
	Description string `json:"description"` // "Грязный гоблин с ржавым ножом"
}

// --- СУЩНОСТЬ ---

type Entity struct {
	// Идентификация
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`

	Pos Position `json:"pos"`

	// Компоненты (Если nil - значит свойство отсутствует)
	Render    *RenderComponent    `json:"render,omitempty"`
	Stats     *StatsComponent     `json:"stats,omitempty"`
	AI        *AIComponent        `json:"ai,omitempty"`
	Narrative *NarrativeComponent `json:"narrative,omitempty"`
}
