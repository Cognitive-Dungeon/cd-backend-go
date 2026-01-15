package ecs

// CommandBuffer позволяет откладывать структурные изменения (Add/Delete).
// Это необходимо для:
// 1. Безопасного добавления компонентов во время итерации (избегает реаллокации).
// 2. Многопоточной записи (каждый поток пишет в свой буфер).
type CommandBuffer struct {
	world *World
	ops   []func(*World)
}

// NewCommandBuffer создает новый буфер, привязанный к миру.
func NewCommandBuffer(w *World) *CommandBuffer {
	return &CommandBuffer{
		world: w,
		ops:   make([]func(*World), 0, 64),
	}
}

// Add откладывает добавление компонента.
// cid — ComponentID, полученный при регистрации.
// val — значение компонента (копируется).
func Add[T any](cb *CommandBuffer, cid int, id EntityID, val T) {
	// Замыкание захватывает значение.
	// Примечание: Это создает аллокацию. Для Logic-слоя это приемлемо.
	cb.ops = append(cb.ops, func(w *World) {
		GetStorage[T](w, cid).Add(id, val)
	})
}

// Delete откладывает удаление компонента.
func Delete[T any](cb *CommandBuffer, cid int, id EntityID) {
	cb.ops = append(cb.ops, func(w *World) {
		GetStorage[T](w, cid).Delete(id)
	})
}

// Execute применяет все отложенные операции к миру.
// Вызывайте это в синхронной точке (между системами).
func (cb *CommandBuffer) Execute() {
	for _, op := range cb.ops {
		op(cb.world)
	}
	// Очищаем список операций для повторного использования
	cb.ops = cb.ops[:0]
}
