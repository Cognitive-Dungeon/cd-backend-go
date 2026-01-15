package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/ecs"
)

type (
	ObjectGuid = types.ObjectGuid
	ObjectType = enums.ObjectType
	Glyph      = types.Glyph
)

const (
	// Размер одного чанка. 1024 элемента * 8 байт (pointer) = 8KB.
	// Это идеально ложится в L1 Cache большинства процессоров.
	ChunkSize = 1024

	// Битовые операции для быстрого деления (вместо / и %)
	ChunkMask  = ChunkSize - 1 // 1023 (0x3FF)
	ChunkShift = 10            // 2^10 = 1024
)

// Instance - Контейнер для всех данных инстанса.
// Хранит состояние конкретного подземелья или континента.
type Instance struct {
	World     *ecs.World
	Grid      *Grid
	nextIndex uint32
}

// NewInstance - создает пустой мир
func NewInstance() *Instance {
	w := ecs.NewWorld()
	inst := &Instance{
		World:     w,
		nextIndex: 1,
	}
	inst.registerComponents()
	return inst
}

func (inst *Instance) registerComponents() {
	// State
	CID_Position = ecs.Register[PositionComponent](inst.World, ecs.ScopePersistent)
	CID_Render = ecs.Register[RenderComponent](inst.World, ecs.ScopePersistent)
	CID_Stats = ecs.Register[StatsComponent](inst.World, ecs.ScopePersistent)
	CID_Name = ecs.Register[NameComponent](inst.World, ecs.ScopePersistent)
	CID_Spellbook = ecs.Register[SpellbookComponent](inst.World, ecs.ScopePersistent)
	CID_Controller = ecs.Register[ControllerComponent](inst.World, ecs.ScopePersistent)

	// Input
	CID_CmdMove = ecs.Register[CmdMove](inst.World, ecs.ScopeInput)
	CID_CmdCast = ecs.Register[CmdCast](inst.World, ecs.ScopeInput)

	// Logic
	CID_IntentMove = ecs.Register[IntentMove](inst.World, ecs.ScopeLogic)
	CID_IntentCast = ecs.Register[IntentCast](inst.World, ecs.ScopeLogic)
}

func (inst *Instance) CreateObject(typ ObjectType) ObjectGuid {
	idx := inst.nextIndex
	inst.nextIndex++

	// Для простоты пока Gen = 1.
	// В продакшене тут нужна логика переиспользования индексов (free list).
	return types.PackObjectGuid(0, uint8(typ), 1, idx)
}

// IsValid проверяет существование сущности.
// В новом ECS мы просто проверяем наличие хоть какого-то компонента или используем ecs.World методы,
// но пока оставим заглушку, т.к. Storage сам проверяет валидность ID при Get.
func (inst *Instance) IsValid(guid ObjectGuid) bool {
	return guid != 0
}

// Helper для конвертации ID
func toECS(guid ObjectGuid) ecs.EntityID {
	return ecs.EntityID(guid) // Прямое приведение, так как битовая структура совпадает
}

type EntityBuilder struct {
	inst *Instance
	id   ecs.EntityID
}

func (inst *Instance) NewEntityBuilder(guid ObjectGuid) *EntityBuilder {
	return &EntityBuilder{
		inst: inst,
		id:   toECS(guid),
	}
}

func (b *EntityBuilder) WithPosition(val PositionComponent) *EntityBuilder {
	ecs.GetStorage[PositionComponent](b.inst.World, CID_Position).Add(b.id, val)
	return b
}

func (b *EntityBuilder) WithStats(val StatsComponent) *EntityBuilder {
	ecs.GetStorage[StatsComponent](b.inst.World, CID_Stats).Add(b.id, val)
	return b
}

func (b *EntityBuilder) WithName(val NameComponent) *EntityBuilder {
	ecs.GetStorage[NameComponent](b.inst.World, CID_Name).Add(b.id, val)
	return b
}

func (b *EntityBuilder) WithRender(g Glyph) *EntityBuilder {
	ecs.GetStorage[RenderComponent](b.inst.World, CID_Render).Add(b.id, RenderComponent{Glyph: g})
	return b
}

func (b *EntityBuilder) WithSpells(val SpellbookComponent) *EntityBuilder {
	ecs.GetStorage[SpellbookComponent](b.inst.World, CID_Spellbook).Add(b.id, val)
	return b
}

func (b *EntityBuilder) WithController(val ControllerComponent) *EntityBuilder {
	ecs.GetStorage[ControllerComponent](b.inst.World, CID_Controller).Add(b.id, val)
	return b
}
