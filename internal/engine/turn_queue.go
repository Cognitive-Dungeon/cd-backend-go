package engine

import (
	"cognitive-server/internal/domain"
	"container/heap"
)

// TurnItem обертка для элемента очереди приоритетов
type TurnItem struct {
	Value    *domain.Entity // Сама сущность
	Priority int            // Приоритет (NextActionTick). Чем меньше, тем раньше ход.
	Index    int            // Индекс в куче (нужен для update)
}

// TurnQueue реализует heap.Interface и хранит TurnItems
type TurnQueue []*TurnItem

func (pq TurnQueue) Len() int { return len(pq) }

func (pq TurnQueue) Less(i, j int) bool {
	// Мы хотим MinHeap, поэтому возвращаем true, если i < j
	return pq[i].Priority < pq[j].Priority
}

func (pq TurnQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *TurnQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*TurnItem)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *TurnQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // избегаем утечки памяти
	item.Index = -1 // для безопасности
	*pq = old[0 : n-1]
	return item
}

// Update изменяет приоритет и значение элемента в очереди
func (pq *TurnQueue) Update(item *TurnItem, priority int) {
	item.Priority = priority
	heap.Fix(pq, item.Index)
}
