package engine

import (
	"cognitive-server/internal/domain"
	"container/heap"
	"testing"
)

func TestTurnQueue(t *testing.T) {
	pq := make(TurnQueue, 0)
	heap.Init(&pq)

	e1 := &domain.Entity{ID: "e1", AI: &domain.AIComponent{NextActionTick: 10}}
	e2 := &domain.Entity{ID: "e2", AI: &domain.AIComponent{NextActionTick: 5}}
	e3 := &domain.Entity{ID: "e3", AI: &domain.AIComponent{NextActionTick: 20}}

	item1 := &TurnItem{Value: e1, Priority: e1.AI.NextActionTick}
	item2 := &TurnItem{Value: e2, Priority: e2.AI.NextActionTick}
	item3 := &TurnItem{Value: e3, Priority: e3.AI.NextActionTick}

	heap.Push(&pq, item1)
	heap.Push(&pq, item2)
	heap.Push(&pq, item3)

	if pq.Len() != 3 {
		t.Errorf("Expected length 3, got %d", pq.Len())
	}

	// First pop should be e2 (Tick 5)
	first := heap.Pop(&pq).(*TurnItem)
	if first.Value.ID != "e2" {
		t.Errorf("Expected e2, got %s", first.Value.ID)
	}

	// Update e1 to be later (Time 10 -> 30)
	// Current queue: e1(10), e3(20). Top is e1.
	// Changing e1 to 30. New Top should be e3.
	pq.Update(item1, 30)

	second := heap.Pop(&pq).(*TurnItem)
	if second.Value.ID != "e3" {
		t.Errorf("Expected e3 (Tick 20), got %s", second.Value.ID)
	}

	third := heap.Pop(&pq).(*TurnItem)
	if third.Value.ID != "e1" {
		t.Errorf("Expected e1 (Tick 30), got %s", third.Value.ID)
	}
}
