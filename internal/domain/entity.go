package domain

import "cognitive-server/internal/core/types"

// --- СУЩНОСТЬ ---

type Entity struct {
	// Идентификация
	ID   types.EntityID `json:"id"`
	Type EntityType     `json:"type"`
	Name string         `json:"name"`

	// ControllerID - ID сессии/пользователя, который управляет этой сущностью.
	// Если пусто - управляется AI.
	ControllerID string `json:"controllerId,omitempty"`

	Pos   Position `json:"pos"`
	Level int      `json:"level"` // Указывает на каком уровне находится сущность

	// Компоненты (Если nil - значит свойство отсутствует)
	Render    *RenderComponent    `json:"render,omitempty"`
	Stats     *StatsComponent     `json:"stats,omitempty"`
	AI        *AIComponent        `json:"ai,omitempty"`
	Narrative *NarrativeComponent `json:"narrative,omitempty"`
	Vision    *VisionComponent    `json:"vision,omitempty"`
	Memory    *MemoryComponent    `json:"memory,omitempty"`
	Trigger   *TriggerComponent   `json:"trigger,omitempty"`

	// Компоненты инвентаря
	Item      *ItemComponent      `json:"item,omitempty"`      // Делает Entity предметом
	Inventory *InventoryComponent `json:"inventory,omitempty"` // Инвентарь для существ
	Equipment *EquipmentComponent `json:"equipment,omitempty"` // Экипировка для существ
}
