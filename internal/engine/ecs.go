package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
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
	// Генерация ID
	freeIndices []uint32 // Стек освободившихся индексов (для переиспользования)
	nextIndex   uint32   // Следующий чистый индекс

	Generations [][]uint16 // Версии слотов (для валидации GUID)
	// Обратная связь: Индекс -> GUID (чтобы проверить валидность)
	Guids [][]ObjectGuid

	// --- Component Storages ---

	Positions [][]*PositionComponent
	Stats     [][]*StatsComponent
	Names     [][]*NameComponent
	Renders   [][]*RenderComponent
	Spells    [][]*SpellbookComponent
	// Casts     map[EntityID]*CastComponent (добавлю позже)
	Controllers [][]*ControllerComponent

	// Данные
	Grid *Grid
}

// NewInstance - создает пустой мир
func NewInstance() *Instance {
	// Создаем первый чанк сразу
	inst := &Instance{
		freeIndices: make([]uint32, 0),
		nextIndex:   0, // Начинаем с 0
	}
	inst.addChunk()
	return inst
}

// addChunk добавляет новую страницу для всех компонентов
func (inst *Instance) addChunk() {
	inst.Generations = append(inst.Generations, make([]uint16, ChunkSize))
	inst.Guids = append(inst.Guids, make([]ObjectGuid, ChunkSize))

	inst.Positions = append(inst.Positions, make([]*PositionComponent, ChunkSize))
	inst.Stats = append(inst.Stats, make([]*StatsComponent, ChunkSize))
	inst.Names = append(inst.Names, make([]*NameComponent, ChunkSize))
	inst.Renders = append(inst.Renders, make([]*RenderComponent, ChunkSize))
	inst.Spells = append(inst.Spells, make([]*SpellbookComponent, ChunkSize))

	inst.Controllers = append(inst.Controllers, make([]*ControllerComponent, ChunkSize))
}

// CreateObject выдает новый ID.
func (inst *Instance) CreateObject(typ ObjectType) ObjectGuid {
	var idx uint32

	// 1. Ищем свободный слот
	if len(inst.freeIndices) > 0 {
		// Pop из стека свободных
		idx = inst.freeIndices[len(inst.freeIndices)-1]
		inst.freeIndices = inst.freeIndices[:len(inst.freeIndices)-1]
	} else {
		// Берем новый
		idx = inst.nextIndex
		inst.nextIndex++

		// Если индекс вылез за пределы текущих чанков — добавляем новый
		// (idx >> ChunkShift) дает индекс нужного чанка
		if int(idx>>ChunkShift) >= len(inst.Guids) {
			inst.addChunk()
		}
	}

	chunkIdx, slotIdx := inst.locate(idx)

	// 2. Инкрементируем поколение (Generation)
	inst.Generations[chunkIdx][slotIdx]++
	gen := inst.Generations[chunkIdx][slotIdx]

	// 3. Собираем GUID (Shard пока 0)
	// TODO: Прокинуть сюда ShardID
	guid := types.PackObjectGuid(0, uint8(typ), gen, idx)

	// 4. Регистрируем
	inst.Guids[chunkIdx][slotIdx] = guid

	return guid

}

func (inst *Instance) clearSlot(chunk, slot uint32) {
	inst.Positions[chunk][slot] = nil
	inst.Stats[chunk][slot] = nil
	inst.Names[chunk][slot] = nil
	inst.Renders[chunk][slot] = nil
	inst.Spells[chunk][slot] = nil
	inst.Controllers[chunk][slot] = nil
}

// DestroyObject удаляет объект (но не стирает память сразу, просто помечает слот)
func (inst *Instance) DestroyObject(guid ObjectGuid) {
	idx := guid.Index()
	chunkIdx, slotIdx := inst.locate(idx)

	// Bounds check (на всякий случай)
	if int(chunkIdx) >= len(inst.Guids) {
		return
	}

	// Валидация: удаляем только если GUID совпадает
	if !inst.Guids[chunkIdx][slotIdx].IsEqual(guid) {
		return
	}

	// Очищаем компоненты (зануляем поинтеры, чтобы GC собрал данные)
	inst.clearSlot(chunkIdx, slotIdx)

	inst.Guids[chunkIdx][slotIdx] = types.NilObjectGuid

	// Возвращаем индекс в пул свободных
	inst.freeIndices = append(inst.freeIndices, idx)
}

// IsValid проверяет жив ли объект по GUID
func (inst *Instance) IsValid(guid ObjectGuid) bool {
	idx := guid.Index()
	chunkIdx, slotIdx := inst.locate(idx)

	if int(chunkIdx) >= len(inst.Guids) {
		return false
	}

	return inst.Guids[chunkIdx][slotIdx].IsEqual(guid)
}

// locate - вычисляет координаты хранения объекта по его глобальному индексу.
//
// chunk — индекс чанка
// slot  — позиция внутри чанка
func (inst *Instance) locate(idx uint32) (chunk, slot uint32) {
	return idx >> ChunkShift, idx & ChunkMask
}

func (inst *Instance) GetPosition(guid ObjectGuid) *PositionComponent {
	if !inst.IsValid(guid) {
		return nil
	}
	chunk, slot := inst.locate(guid.Index())
	return inst.Positions[chunk][slot]
}

func (inst *Instance) GetStats(guid ObjectGuid) *StatsComponent {
	if !inst.IsValid(guid) {
		return nil
	}
	chunk, slot := inst.locate(guid.Index())
	return inst.Stats[chunk][slot]
}

func (inst *Instance) GetName(guid ObjectGuid) *NameComponent {
	if !inst.IsValid(guid) {
		return nil
	}
	chunk, slot := inst.locate(guid.Index())
	return inst.Names[chunk][slot]
}

func (inst *Instance) GetRender(guid ObjectGuid) *RenderComponent {
	if !inst.IsValid(guid) {
		return nil
	}
	chunk, slot := inst.locate(guid.Index())
	return inst.Renders[chunk][slot]
}

func (inst *Instance) GetSpells(guid ObjectGuid) *SpellbookComponent {
	if !inst.IsValid(guid) {
		return nil
	}
	chunk, slot := inst.locate(guid.Index())
	return inst.Spells[chunk][slot]
}

func (inst *Instance) GetController(guid ObjectGuid) *ControllerComponent {
	if !inst.IsValid(guid) {
		return nil
	}
	chunk, slot := inst.locate(guid.Index())
	return inst.Controllers[chunk][slot]
}

type EntityBuilder struct {
	inst  *Instance
	id    ObjectGuid
	chunk uint32
	slot  uint32
}

func (inst *Instance) NewEntityBuilder(id ObjectGuid) *EntityBuilder {
	chunk, slot := inst.locate(id.Index())
	return &EntityBuilder{
		inst:  inst,
		id:    id,
		chunk: chunk,
		slot:  slot,
	}
}

func (b *EntityBuilder) WithPosition(pos PositionComponent) *EntityBuilder {
	b.inst.Positions[b.chunk][b.slot] = &pos
	return b
}

func (b *EntityBuilder) WithStats(stats StatsComponent) *EntityBuilder {
	b.inst.Stats[b.chunk][b.slot] = &stats
	return b
}

func (b *EntityBuilder) WithName(name NameComponent) *EntityBuilder {
	b.inst.Names[b.chunk][b.slot] = &name
	return b
}

func (b *EntityBuilder) WithRender(g Glyph) *EntityBuilder {
	b.inst.Renders[b.chunk][b.slot] = &RenderComponent{
		Glyph: g,
	}
	return b
}

func (b *EntityBuilder) WithSpells(spells SpellbookComponent) *EntityBuilder {
	b.inst.Spells[b.chunk][b.slot] = &spells
	return b
}

func (b *EntityBuilder) WithController(ctrl ControllerComponent) *EntityBuilder {
	b.inst.Controllers[b.chunk][b.slot] = &ctrl
	return b
}
