package domain

import "errors"

func (w *GameWorld) GetIndex(x, y int) int {
	return y*w.Width + x
}

// GetEntitiesAt возвращает список сущностей в конкретной клетке (быстро!)
func (w *GameWorld) GetEntitiesAt(x, y int) []*Entity {
	if x < 0 || x >= w.Width || y < 0 || y >= w.Height {
		return nil
	}
	idx := w.GetIndex(x, y)
	return w.SpatialHash[idx]
}

// GetEntity ищет сущность по ID
func (w *GameWorld) GetEntity(id string) *Entity {
	if w.EntityRegistry == nil {
		return nil
	}
	return w.EntityRegistry[id]
}

// RegisterEntity добавляет сущность в реестр
func (w *GameWorld) RegisterEntity(e *Entity) {
	if w.EntityRegistry == nil {
		w.EntityRegistry = make(map[string]*Entity)
	}
	w.EntityRegistry[e.ID] = e
}

// UnregisterEntity удаляет сущность из реестра
func (w *GameWorld) UnregisterEntity(id string) {
	if w.EntityRegistry != nil {
		delete(w.EntityRegistry, id)
	}
}

// AddEntity добавляет сущность в индекс
func (w *GameWorld) AddEntity(e *Entity) {
	idx := w.GetIndex(e.Pos.X, e.Pos.Y)
	w.SpatialHash[idx] = append(w.SpatialHash[idx], e)
}

// RemoveEntity удаляет сущность из индекса (например, при смерти или телепорте)
func (w *GameWorld) RemoveEntity(e *Entity) {
	idx := w.GetIndex(e.Pos.X, e.Pos.Y)
	entities := w.SpatialHash[idx]

	for i, other := range entities {
		if other.ID == e.ID {
			// Удаляем элемент из слайса (быстрый способ без сохранения порядка)
			// w.SpatialHash[idx] = append(entities[:i], entities[i+1:]...)

			// Более оптимальный способ (Swap with last), если порядок не важен:
			lastIdx := len(entities) - 1
			entities[i] = entities[lastIdx]
			w.SpatialHash[idx] = entities[:lastIdx]
			return
		}
	}
}

// UpdateEntityPos перемещает сущность в индексе
func (w *GameWorld) UpdateEntityPos(e *Entity, newX, newY int) error {
	// 1. Проверка границ (на всякий случай)
	if newX < 0 || newX >= w.Width || newY < 0 || newY >= w.Height {
		return errors.New("out of bounds")
	}

	// 2. Удаляем из старой позиции
	w.RemoveEntity(e)

	// 3. Обновляем координаты в сущности
	e.Pos.X = newX
	e.Pos.Y = newY

	// 4. Добавляем в новую позицию
	w.AddEntity(e)
	return nil
}
