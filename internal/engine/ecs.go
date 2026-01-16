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
	CID_Position = ecs.RegisterState[PositionComponent](inst.World)
	CID_Render = ecs.RegisterState[RenderComponent](inst.World)
	CID_Stats = ecs.RegisterState[StatsComponent](inst.World)
	CID_Name = ecs.RegisterState[NameComponent](inst.World)
	CID_Spellbook = ecs.RegisterState[SpellbookComponent](inst.World)
	CID_Controller = ecs.RegisterState[ControllerComponent](inst.World)

	// Input
	CID_CmdMove = ecs.RegisterInput[CmdMove](inst.World)
	CID_CmdCast = ecs.RegisterInput[CmdCast](inst.World)

	// Logic
	CID_IntentMove = ecs.RegisterLogic[IntentMove](inst.World)
	CID_IntentCast = ecs.RegisterLogic[IntentCast](inst.World)
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

func (b *EntityBuilder) WithPosition(x, y int) *EntityBuilder {
	comp := PositionComponent{
		TilePos: types.TilePos{X: types.TileCoord(x), Y: types.TileCoord(y)},
	}
	ecs.GetStorage[PositionComponent](b.inst.World, CID_Position).Add(b.id, comp)
	return b
}

func (b *EntityBuilder) WithStats(hp, mana int32) *EntityBuilder {
	comp := StatsComponent{
		Health:    hp,
		MaxHealth: hp,
		Mana:      mana,
		MaxMana:   mana,
	}
	ecs.GetStorage[StatsComponent](b.inst.World, CID_Stats).Add(b.id, comp)
	return b
}

func (b *EntityBuilder) WithName(name string) *EntityBuilder {
	ecs.GetStorage[NameComponent](b.inst.World, CID_Name).Add(b.id, NameComponent{Name: name})
	return b
}

func (b *EntityBuilder) WithRender(char byte, color uint32) *EntityBuilder {
	comp := RenderComponent{
		Glyph: types.MakeGlyph(color, char),
	}
	ecs.GetStorage[RenderComponent](b.inst.World, CID_Render).Add(b.id, comp)
	return b
}

func (b *EntityBuilder) WithSpells(spells ...uint32) *EntityBuilder {
	comp := SpellbookComponent{
		KnownSpells: spells,
		Cooldowns:   make(map[uint32]float64),
		GCD:         0,
	}
	ecs.GetStorage[SpellbookComponent](b.inst.World, CID_Spellbook).Add(b.id, comp)
	return b
}

func (b *EntityBuilder) WithController(agentID string) *EntityBuilder {
	ecs.GetStorage[ControllerComponent](b.inst.World, CID_Controller).Add(b.id, ControllerComponent{AgentID: agentID})
	return b
}
