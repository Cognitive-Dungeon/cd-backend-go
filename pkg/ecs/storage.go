package ecs

// AnyStorage — стирающий тип интерфейс для хранения любых компонентов.
// Используется Миром для управления жизненным циклом (Clear, Delete).
type AnyStorage interface {
	// Clear очищает хранилище (сохраняя capacity).
	Clear()
	// Delete удаляет компонент по EntityID.
	Delete(id EntityID)
}

// Storage реализует паттерн Sparse Set для типа T.
// Гарантирует O(1) добавление, удаление и доступ.
// Данные хранятся в плотном массиве (dense), что обеспечивает Cache Locality.
type Storage[T any] struct {
	// values хранит сами компоненты. Итерируемся по этому массиву.
	values []T

	// ids хранит полные EntityID сущностей, соответствующих элементам values.
	// Используется для проверки Generation при доступе (защита от ABA).
	ids []EntityID

	// sparse мапит Index сущности на индекс в массиве values.
	// sparse[entityIndex] -> denseIndex.
	// Значение -1 означает отсутствие компонента.
	sparse []int32
}

// NewStorage создает новое хранилище с начальной емкостью.
func NewStorage[T any]() *Storage[T] {
	return &Storage[T]{
		values: make([]T, 0, 1024),
		ids:    make([]EntityID, 0, 1024),
		sparse: make([]int32, 0),
	}
}

// Add добавляет компонент или перезаписывает существующий.
// Возвращает указатель на данные внутри слайса для инициализации.
//
// ВНИМАНИЕ: Возвращенный указатель валиден только до следующего добавления
// в этот же Storage (из-за возможной реаллокации слайса).
func (s *Storage[T]) Add(id EntityID, val T) *T {
	idx := id.Index()
	s.ensureSparse(idx)

	// Проверяем, занят ли слот
	if denseIdx := s.sparse[idx]; denseIdx != -1 {
		// Слот занят. Это может быть та же сущность или старая (мертвая).
		// В обоих случаях мы перезаписываем данные новой версией.

		s.values[denseIdx] = val
		s.ids[denseIdx] = id // Обновляем поколение EntityID
		return &s.values[denseIdx]
	}

	// Добавляем в конец (Append)
	denseIdx := int32(len(s.values))
	s.sparse[idx] = denseIdx

	s.ids = append(s.ids, id)
	s.values = append(s.values, val)

	return &s.values[denseIdx]
}

// Get возвращает указатель на компонент.
// Если компонента нет или поколение EntityID не совпадает (сущность переродилась), возвращает nil.
func (s *Storage[T]) Get(id EntityID) *T {
	idx := id.Index()

	// 1. Проверка границ sparse массива
	if int(idx) >= len(s.sparse) {
		return nil
	}

	// 2. Проверка наличия компонента
	denseIdx := s.sparse[idx]
	if denseIdx == -1 {
		return nil
	}

	// 3. ABA Protection: Сравниваем поколения.
	// Если сохраненный EntityID не равен запрошенному, значит слот принадлежит
	// другой сущности (с тем же индексом, но новым поколением).
	if s.ids[denseIdx] != id {
		return nil
	}

	return &s.values[denseIdx]
}

// Delete удаляет компонент, используя технику Swap & Pop.
// Это O(1) операция, которая не сохраняет порядок элементов.
func (s *Storage[T]) Delete(id EntityID) {
	idx := id.Index()
	if int(idx) >= len(s.sparse) {
		return
	}

	denseIdx := s.sparse[idx]
	if denseIdx == -1 {
		return
	}

	// ABA Protection при удалении:
	// Не удаляем чужой компонент, если индекс переиспользован.
	if s.ids[denseIdx] != id {
		return
	}

	lastIdx := len(s.values) - 1

	// Если удаляемый элемент не последний, меняем его местами с последним
	if int(denseIdx) != lastIdx {
		lastID := s.ids[lastIdx]

		// Переносим данные
		s.values[denseIdx] = s.values[lastIdx]
		s.ids[denseIdx] = lastID

		// Обновляем ссылку в sparse массиве для перемещенной сущности
		s.sparse[lastID.Index()] = denseIdx
	}

	// Удаляем последний элемент (уменьшаем len, сохраняем cap)
	s.values = s.values[:lastIdx]
	s.ids = s.ids[:lastIdx]

	// Помечаем слот в sparse как пустой
	s.sparse[idx] = -1
}

// Clear полностью очищает хранилище.
// Используется для временных компонентов (ScopeInput, ScopeLogic и т.д.).
// Операция очень быстрая, так как не освобождает память (capacity сохраняется).
func (s *Storage[T]) Clear() {
	// Быстрая инвалидация sparse ссылок
	for _, id := range s.ids {
		s.sparse[id.Index()] = -1
	}

	// Сброс длины слайсов
	s.values = s.values[:0]
	s.ids = s.ids[:0]
}

// Slice возвращает прямой доступ к массиву значений.
// ВНИМАНИЕ: Порядок элементов произвольный и не соответствует индексам сущностей.
// Используйте это для SIMD-оптимизаций или простой итерации "по всем".
func (s *Storage[T]) Slice() []T {
	return s.values
}

// ensureSparse расширяет sparse массив при необходимости.
func (s *Storage[T]) ensureSparse(idx uint32) {
	if int(idx) < len(s.sparse) {
		return
	}

	// Стратегия роста: увеличиваем минимум в 1.5 раза или до нужного индекса
	currCap := len(s.sparse)
	targetCap := int(idx) + 1

	if targetCap < currCap*2 {
		targetCap = currCap * 2
	}
	if targetCap < 1024 {
		targetCap = 1024
	}

	newSparse := make([]int32, targetCap)
	copy(newSparse, s.sparse)

	// Заполняем новые слоты значением -1
	for i := len(s.sparse); i < len(newSparse); i++ {
		newSparse[i] = -1
	}

	s.sparse = newSparse
}
