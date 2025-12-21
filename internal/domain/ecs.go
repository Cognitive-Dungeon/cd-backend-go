package domain

import (
	"reflect"
)

// ComponentStorage — интерфейс для стирания типа (type erasure),
// чтобы хранить разные Storage[T] в одной мапе.
type ComponentStorage interface {
	Delete(id EntityID)
}

// Storage — типизированное хранилище для конкретного компонента T.
type Storage[T any] struct {
	data map[EntityID]T
}

func NewStorage[T any]() *Storage[T] {
	return &Storage[T]{
		data: make(map[EntityID]T),
	}
}

func (s *Storage[T]) Set(id EntityID, val T) {
	s.data[id] = val
}

func (s *Storage[T]) Get(id EntityID) (T, bool) {
	val, ok := s.data[id]
	return val, ok
}

func (s *Storage[T]) Delete(id EntityID) {
	delete(s.data, id)
}

// WorldComponents — менеджер всех компонентов мира.
// Это "мешок", в котором лежат все мапы компонентов.
type WorldComponents struct {
	// registry хранит мапу: reflect.Type -> ComponentStorage
	registry map[reflect.Type]ComponentStorage
}

func NewWorldComponents() *WorldComponents {
	return &WorldComponents{
		registry: make(map[reflect.Type]ComponentStorage),
	}
}

// getStorage (private) — магия рефлексии для получения нужной мапы.
// Работает быстро, так как reflect.TypeOf кэшируется рантаймом Go.
func getStorage[T any](wc *WorldComponents) *Storage[T] {
	t := reflect.TypeOf((*T)(nil)).Elem()

	storage, exists := wc.registry[t]
	if !exists {
		storage = NewStorage[T]()
		wc.registry[t] = storage
	}
	return storage.(*Storage[T])
}

// --- PUBLIC API ---

// SetComponent сохраняет компонент T для сущности ID.
func SetComponent[T any](wc *WorldComponents, id EntityID, comp T) {
	getStorage[T](wc).Set(id, comp)
}

// GetComponent получает компонент T для сущности ID.
func GetComponent[T any](wc *WorldComponents, id EntityID) (T, bool) {
	return getStorage[T](wc).Get(id)
}

// RemoveEntityComponents удаляет ВСЕ редкие компоненты для этой сущности.
// Это нужно вызывать при смерти или удалении объекта.
func (wc *WorldComponents) RemoveEntityComponents(id EntityID) {
	for _, store := range wc.registry {
		store.Delete(id)
	}
}
