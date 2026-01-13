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
func (w *Instance) addChunk() {
	w.Generations = append(w.Generations, make([]uint16, ChunkSize))
	w.Guids = append(w.Guids, make([]ObjectGuid, ChunkSize))

	w.Positions = append(w.Positions, make([]*PositionComponent, ChunkSize))
	w.Stats = append(w.Stats, make([]*StatsComponent, ChunkSize))
	w.Names = append(w.Names, make([]*NameComponent, ChunkSize))
	w.Renders = append(w.Renders, make([]*RenderComponent, ChunkSize))
	w.Spells = append(w.Spells, make([]*SpellbookComponent, ChunkSize))

	w.Controllers = append(w.Controllers, make([]*ControllerComponent, ChunkSize))
}

// CreateObject выдает новый ID.
func (w *Instance) CreateObject(typ ObjectType) ObjectGuid {
	var idx uint32

	// 1. Ищем свободный слот
	if len(w.freeIndices) > 0 {
		// Pop из стека свободных
		idx = w.freeIndices[len(w.freeIndices)-1]
		w.freeIndices = w.freeIndices[:len(w.freeIndices)-1]
	} else {
		// Берем новый
		idx = w.nextIndex
		w.nextIndex++

		// Если индекс вылез за пределы текущих чанков — добавляем новый
		// (idx >> ChunkShift) дает индекс нужного чанка
		if int(idx>>ChunkShift) >= len(w.Guids) {
			w.addChunk()
		}
	}

	chunkIdx, slotIdx := w.locate(idx)

	// 2. Инкрементируем поколение (Generation)
	w.Generations[chunkIdx][slotIdx]++
	gen := w.Generations[chunkIdx][slotIdx]

	// 3. Собираем GUID (Shard пока 0)
	// TODO: Прокинуть сюда ShardID
	guid := types.PackObjectGuid(0, uint8(typ), gen, idx)

	// 4. Регистрируем
	w.Guids[chunkIdx][slotIdx] = guid

	return guid

}

func (w *Instance) clearSlot(chunk, slot uint32) {
	w.Positions[chunk][slot] = nil
	w.Stats[chunk][slot] = nil
	w.Names[chunk][slot] = nil
	w.Renders[chunk][slot] = nil
	w.Spells[chunk][slot] = nil
	w.Controllers[chunk][slot] = nil
}

// DestroyObject удаляет объект (но не стирает память сразу, просто помечает слот)
func (w *Instance) DestroyObject(guid ObjectGuid) {
	idx := guid.Index()
	chunkIdx, slotIdx := w.locate(idx)

	// Bounds check (на всякий случай)
	if int(chunkIdx) >= len(w.Guids) {
		return
	}

	// Валидация: удаляем только если GUID совпадает
	if !w.Guids[chunkIdx][slotIdx].IsEqual(guid) {
		return
	}

	// Очищаем компоненты (зануляем поинтеры, чтобы GC собрал данные)
	w.clearSlot(chunkIdx, slotIdx)

	w.Guids[chunkIdx][slotIdx] = types.NilObjectGuid

	// Возвращаем индекс в пул свободных
	w.freeIndices = append(w.freeIndices, idx)
}

// IsValid проверяет жив ли объект по GUID
func (w *Instance) IsValid(guid ObjectGuid) bool {
	idx := guid.Index()
	chunkIdx, slotIdx := w.locate(idx)

	if int(chunkIdx) >= len(w.Guids) {
		return false
	}

	return w.Guids[chunkIdx][slotIdx].IsEqual(guid)
}

// locate - вычисляет координаты хранения объекта по его глобальному индексу.
//
// chunk — индекс чанка
// slot  — позиция внутри чанка
func (w *Instance) locate(idx uint32) (chunk, slot uint32) {
	return idx >> ChunkShift, idx & ChunkMask
}

func (w *Instance) GetPosition(guid ObjectGuid) *PositionComponent {
	if !w.IsValid(guid) {
		return nil
	}
	chunk, slot := w.locate(guid.Index())
	return w.Positions[chunk][slot]
}

func (w *Instance) GetStats(guid ObjectGuid) *StatsComponent {
	if !w.IsValid(guid) {
		return nil
	}
	chunk, slot := w.locate(guid.Index())
	return w.Stats[chunk][slot]
}

func (w *Instance) GetName(guid ObjectGuid) *NameComponent {
	if !w.IsValid(guid) {
		return nil
	}
	chunk, slot := w.locate(guid.Index())
	return w.Names[chunk][slot]
}

func (w *Instance) GetRender(guid ObjectGuid) *RenderComponent {
	if !w.IsValid(guid) {
		return nil
	}
	chunk, slot := w.locate(guid.Index())
	return w.Renders[chunk][slot]
}

func (w *Instance) GetSpells(guid ObjectGuid) *SpellbookComponent {
	if !w.IsValid(guid) {
		return nil
	}
	chunk, slot := w.locate(guid.Index())
	return w.Spells[chunk][slot]
}

func (w *Instance) GetController(guid ObjectGuid) *ControllerComponent {
	if !w.IsValid(guid) {
		return nil
	}
	chunk, slot := w.locate(guid.Index())
	return w.Controllers[chunk][slot]
}

type EntityBuilder struct {
	inst  *Instance
	id    ObjectGuid
	chunk uint32
	slot  uint32
}

func (w *Instance) NewEntityBuilder(id ObjectGuid) *EntityBuilder {
	chunk, slot := w.locate(id.Index())
	return &EntityBuilder{
		inst:  w,
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
