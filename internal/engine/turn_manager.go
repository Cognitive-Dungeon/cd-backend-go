package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/logger"
	"container/heap"
)

// TurnManager manages the priority queue of entity turns.
type TurnManager struct {
	queue   TurnQueue
	itemMap map[domain.EntityID]*TurnItem
}

func NewTurnManager() *TurnManager {
	return &TurnManager{
		queue:   make(TurnQueue, 0),
		itemMap: make(map[domain.EntityID]*TurnItem),
	}
}

// AddEntity registers an entity in the turn system.
func (tm *TurnManager) AddEntity(e *domain.Entity) {
	if e.AI == nil {
		return
	}

	// Create queue item
	item := &TurnItem{
		Value:    e,
		Priority: e.AI.NextActionTick,
	}

	heap.Push(&tm.queue, item)
	tm.itemMap[e.ID] = item

	logger.Log.WithField("entity_id", e.ID).Debug("Entity added to TurnManager")
}

// UpdatePriority updates an entity's position in the queue (e.g. after they acted).
func (tm *TurnManager) UpdatePriority(entityID domain.EntityID, newTick int) {
	if item, ok := tm.itemMap[entityID]; ok {
		tm.queue.Update(item, newTick)
	}
}

// PeekNext returns the entity whose turn is next, without removing them.
func (tm *TurnManager) PeekNext() *TurnItem {
	if tm.queue.Len() == 0 {
		return nil
	}
	return tm.queue[0]
}

// RemoveEntity removed an entity from the turn system (e.g. death).
func (tm *TurnManager) RemoveEntity(entityID domain.EntityID) {
	if item, ok := tm.itemMap[entityID]; ok {
		heap.Remove(&tm.queue, item.Index)
		delete(tm.itemMap, entityID)
	}
}

func (tm *TurnManager) Len() int {
	return tm.queue.Len()
}

// DebugDump возвращает снимок очереди для отладки
func (tm *TurnManager) DebugDump() []map[string]interface{} {
	// Инициализируем как пустой слайс, а не nil. Тогда в JSON это будет "[]", а не "null"
	result := make([]map[string]interface{}, 0)

	for _, item := range tm.queue {
		result = append(result, map[string]interface{}{
			"id":       item.Value.ID,
			"name":     item.Value.Name,
			"priority": item.Priority,
			"index":    item.Index,
		})
	}
	return result
}
