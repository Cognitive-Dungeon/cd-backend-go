package domain

// TransitionType описывает механику перехода
type TransitionType uint8

const (
	TransitionTypeUnknown TransitionType = iota
	// Лестница (требует ответной лестницы)
	TransitionTypeStairs
	// Дыра (падаешь в случайное место или под себя)
	TransitionTypeHole
	// Магический портал (ищет другой портал)
	TransitionTypePortal
)

// TransitionDirection описывает вектор
type TransitionDirection int8

const (
	TransitionDirectionUp   TransitionDirection = -1
	TransitionDirectionSame TransitionDirection = 0
	TransitionDirectionDown TransitionDirection = 1
)

// TransitionComponent — Компонент вертикального перемещения
type TransitionComponent struct {
	Type      TransitionType      `json:"type"`
	Direction TransitionDirection `json:"dir"` // +1, -1
}
