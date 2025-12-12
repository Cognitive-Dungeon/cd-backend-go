package domain

import "errors"

// --- REGISTRY (Поиск по ID) ---

func (w *GameWorld) GetEntity(id string) *Entity {
	if w.EntityRegistry == nil {
		return nil
	}
	return w.EntityRegistry[id]
}

func (w *GameWorld) RegisterEntity(e *Entity) {
	if w.EntityRegistry == nil {
		w.EntityRegistry = make(map[string]*Entity)
	}
	w.EntityRegistry[e.ID] = e
}

func (w *GameWorld) UnregisterEntity(id string) {
	if w.EntityRegistry != nil {
		delete(w.EntityRegistry, id)
	}
}

// --- SPATIAL INDEX (Позиция на карте) ---

func (w *GameWorld) GetIndex(x, y int) int {
	return y*w.Width + x
}

func (w *GameWorld) GetEntitiesAt(x, y int) []*Entity {
	if x < 0 || x >= w.Width || y < 0 || y >= w.Height {
		return nil
	}
	idx := w.GetIndex(x, y)
	return w.SpatialHash[idx]
}

// AddEntity добавляет сущность ТОЛЬКО в пространственный индекс.
// Не влияет на Registry.
func (w *GameWorld) AddEntity(e *Entity) {
	idx := w.GetIndex(e.Pos.X, e.Pos.Y)
	w.SpatialHash[idx] = append(w.SpatialHash[idx], e)
}

// removeFromSpatial удаляет сущность ТОЛЬКО из пространственного индекса.
// Это приватный метод, чтобы случайно не забыть про Registry при полном удалении.
func (w *GameWorld) removeFromSpatial(e *Entity) {
	idx := w.GetIndex(e.Pos.X, e.Pos.Y)
	entities := w.SpatialHash[idx]

	for i, other := range entities {
		if other.ID == e.ID {
			// Удаляем из слайса (Swap with last для скорости)
			lastIdx := len(entities) - 1
			entities[i] = entities[lastIdx]
			entities[lastIdx] = nil
			w.SpatialHash[idx] = entities[:lastIdx]
			return
		}
	}
}

// --- HIGH LEVEL ACTIONS (Комбинации) ---

// RemoveEntity полностью удаляет сущность из мира (Смерть, Подбор в инвентарь).
// Удаляет И из карты, И из реестра.
func (w *GameWorld) RemoveEntity(e *Entity) {
	w.removeFromSpatial(e)
	w.UnregisterEntity(e.ID)
}

// UpdateEntityPos обрабатывает перемещение.
// Работает ТОЛЬКО с пространственным индексом. Реестр не трогает.
func (w *GameWorld) UpdateEntityPos(e *Entity, newX, newY int) error {
	// 1. Проверка границ
	if newX < 0 || newX >= w.Width || newY < 0 || newY >= w.Height {
		return errors.New("out of bounds")
	}

	// 2. Убираем со старой клетки (только Spatial!)
	w.removeFromSpatial(e)

	// 3. Меняем координаты
	e.Pos.X = newX
	e.Pos.Y = newY

	// 4. Ставим на новую клетку (только Spatial!)
	w.AddEntity(e)

	return nil
}
