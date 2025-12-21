package domain

import (
	"reflect"
)

// ComponentStorage — интерфейс, стирающий тип T, чтобы хранить всё в одной мапе
type ComponentStorage interface {
	Delete(id EntityID)
}

// Storage — типизированное хранилище конкретного компонента T
type Storage[T any] struct {
	Data map[EntityID]T
}

func NewStorage[T any]() *Storage[T] {
	return &Storage[T]{
		Data: make(map[EntityID]T),
	}
}

func (s *Storage[T]) Set(id EntityID, val T) {
	s.Data[id] = val
}

func (s *Storage[T]) Get(id EntityID) (T, bool) {
	val, ok := s.Data[id]
	return val, ok
}

func (s *Storage[T]) Delete(id EntityID) {
	delete(s.Data, id)
}

// WorldComponents — контейнер для всех типов компонентов мира
type WorldComponents struct {
	// registry хранит мапу: reflect.Type -> ComponentStorage
	registry map[reflect.Type]ComponentStorage
}

func NewWorldComponents() *WorldComponents {
	return &WorldComponents{
		registry: make(map[reflect.Type]ComponentStorage),
	}
}

// GetStorage возвращает (или создает) хранилище для типа T.
// Public, так как нужен Системам для итерации (range storage.Data).
func GetStorage[T any](wc *WorldComponents) *Storage[T] {
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
	GetStorage[T](wc).Set(id, comp)
}

// GetComponent получает компонент T для сущности ID.
func GetComponent[T any](wc *WorldComponents, id EntityID) (T, bool) {
	return GetStorage[T](wc).Get(id)
}

func RemoveComponent[T any](wc *WorldComponents, id EntityID) {
	GetStorage[T](wc).Delete(id)
}

// RemoveEntityComponents удаляет ВСЕ редкие компоненты для этой сущности.
// Это нужно вызывать при смерти или удалении объекта.
func (wc *WorldComponents) RemoveEntityComponents(id EntityID) {
	for _, store := range wc.registry {
		store.Delete(id)
	}
}
