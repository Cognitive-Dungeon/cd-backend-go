package engine

import (
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
)

// --- INPUT PHASE CONTRACT ---

// InputContext передается только в системы ввода.
// Его задача — предоставить инструменты для чтения команд и создания намерений.
type InputContext struct {
	World         *ecs.World
	Commands      *ecs.CommandBuffer // Обязателен для создания Intent
	SpellRegistry *SpellRegistry     // Нужен для проверки CmdCast
}

// NewInputContext создает контекст и автоматически инициализирует буфер команд.
func NewInputContext(w *ecs.World, reg *SpellRegistry) InputContext {
	return InputContext{
		World:         w,
		Commands:      ecs.NewCommandBuffer(w),
		SpellRegistry: reg,
	}
}

// Commit применяет все команды, накопленные системами ввода.
func (ctx InputContext) Commit() {
	ctx.Commands.Execute()
}

// InputSystemFunc — сигнатура функции системы ввода.
type InputSystemFunc func(ctx InputContext)

// --- LOGIC PHASE CONTRACT ---

// LogicContext передается только в системы логики.
// Его задача — предоставить доступ к изменению мира и реакциям.
type LogicContext struct {
	World         *ecs.World
	Grid          *Grid              // Нужен для проверки стен
	Bus           *eventbus.EventBus // Нужен для событий (урон, лог)
	SpellRegistry *SpellRegistry     // Нужен для свойств спеллов
}

// NewLogicContext собирает зависимости для логического шага.
func NewLogicContext(w *ecs.World, grid *Grid, bus *eventbus.EventBus, reg *SpellRegistry) LogicContext {
	return LogicContext{
		World:         w,
		Grid:          grid,
		Bus:           bus,
		SpellRegistry: reg,
	}
}

// LogicSystemFunc — сигнатура функции системы логики.
type LogicSystemFunc func(ctx LogicContext)
