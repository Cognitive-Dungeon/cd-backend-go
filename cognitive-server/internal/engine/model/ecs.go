package model

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/entityindex"
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/grid"
	"cognitive-server/pkg/types/glyph"
	"cognitive-server/pkg/worldmap"
)

type (
	ObjectGuid = types.ObjectGuid
	ObjectType = enums.ObjectType
	Glyph      = glyph.Glyph
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
	World      *ecs.World
	WorldMap   *worldmap.World
	EntityGrid *entityindex.Grid
	nextIndex  uint32
}

// NewInstance - создает пустой мир
func NewInstance() *Instance {
	w := ecs.NewWorld()
	inst := &Instance{
		World:      w,
		WorldMap:   worldmap.NewWorld(),
		EntityGrid: entityindex.New(),
		nextIndex:  1,
	}
	inst.registerComponents()
	return inst
}

func (inst *Instance) registerComponents() {
	// State
	components.CID_Position = ecs.RegisterState[components.PositionComponent](inst.World)
	components.CID_Render = ecs.RegisterState[components.RenderComponent](inst.World)
	components.CID_Stats = ecs.RegisterState[components.StatsComponent](inst.World)
	components.CID_Name = ecs.RegisterState[components.NameComponent](inst.World)
	components.CID_Spellbook = ecs.RegisterState[components.SpellbookComponent](inst.World)
	components.CID_Controller = ecs.RegisterState[components.ControllerComponent](inst.World)

	// Input
	components.CID_CmdMove = ecs.RegisterInput[components.CmdMove](inst.World)
	components.CID_CmdCast = ecs.RegisterInput[components.CmdCast](inst.World)

	// Logic
	components.CID_IntentMove = ecs.RegisterLogic[components.IntentMove](inst.World)
	components.CID_IntentCast = ecs.RegisterLogic[components.IntentCast](inst.World)
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
	pos := grid.TilePos{X: grid.TileCoord(x), Y: grid.TileCoord(y)}
	comp := components.PositionComponent{
		TilePos: pos,
	}
	ecs.GetStorage[components.PositionComponent](b.inst.World, components.CID_Position).Add(b.id, comp)

	geoPos := geo.Pos(x, y, 0)
	b.inst.EntityGrid.Add(b.id, geoPos)

	return b
}

func (b *EntityBuilder) WithStats(hp, mana int32) *EntityBuilder {
	comp := components.StatsComponent{
		Health:    hp,
		MaxHealth: hp,
		Mana:      mana,
		MaxMana:   mana,
	}
	ecs.GetStorage[components.StatsComponent](b.inst.World, components.CID_Stats).Add(b.id, comp)
	return b
}

func (b *EntityBuilder) WithName(name string) *EntityBuilder {
	ecs.GetStorage[components.NameComponent](b.inst.World, components.CID_Name).Add(b.id, components.NameComponent{Name: name})
	return b
}

func (b *EntityBuilder) WithRender(char byte, color uint32) *EntityBuilder {
	comp := components.RenderComponent{
		Glyph: glyph.MakeGlyph(color, char),
	}
	ecs.GetStorage[components.RenderComponent](b.inst.World, components.CID_Render).Add(b.id, comp)
	return b
}

func (b *EntityBuilder) WithSpells(spells ...uint32) *EntityBuilder {
	comp := components.SpellbookComponent{
		KnownSpells: spells,
		Cooldowns:   make(map[uint32]float64),
		GCD:         0,
	}
	ecs.GetStorage[components.SpellbookComponent](b.inst.World, components.CID_Spellbook).Add(b.id, comp)
	return b
}

func (b *EntityBuilder) WithController(agentID string) *EntityBuilder {
	ecs.GetStorage[components.ControllerComponent](b.inst.World, components.CID_Controller).Add(b.id, components.ControllerComponent{AgentID: agentID})
	return b
}
