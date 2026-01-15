package ecs

import "reflect"

// World — главный контейнер ECS.
// Хранит все пулы компонентов и управляет их очисткой.
type World struct {
	// storages хранит пулы. Доступ по индексу (ComponentID).
	storages []AnyStorage

	// typeToIndex мапит Go-тип (reflect.Type) в индекс массива storages.
	typeToIndex map[reflect.Type]int

	// transients хранит списки стореджей, которые нужно очищать
	// на определенных этапах кадра.
	transients map[Scope][]AnyStorage
}

// NewWorld создает новый пустой мир.
func NewWorld() *World {
	return &World{
		storages:    make([]AnyStorage, 0, 64),
		typeToIndex: make(map[reflect.Type]int),
		transients:  make(map[Scope][]AnyStorage),
	}
}

// Register региструет тип компонента T в мире и возвращает его уникальный EntityID.
//
// Параметр scope определяет, когда данные этого типа будут автоматически очищены.
// Вызывайте эту функцию при инициализации приложения.
func Register[T any](w *World, scope Scope) int {
	typ := reflect.TypeOf((*T)(nil)).Elem()

	// Защита от дубликатов
	if _, exists := w.typeToIndex[typ]; exists {
		panic("ecs: component already registered: " + typ.Name())
	}

	id := len(w.storages)
	s := NewStorage[T]()

	w.storages = append(w.storages, s)
	w.typeToIndex[typ] = id

	// Если компонент временный, добавляем его в списки очистки
	if scope != ScopePersistent {
		w.addToCleanupList(s, scope)
	}

	return id
}

// GetStorage возвращает типизированный пул для компонента по его EntityID.
// Это самый быстрый способ доступа (доступ по индексу массива).
//
// id должен быть получен из функции Register.
func GetStorage[T any](w *World, storageID int) *Storage[T] {
	return w.storages[storageID].(*Storage[T])
}

// GetStorageByType — медленный доступ через reflect.
func GetStorageByType[T any](w *World) *Storage[T] {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	id, ok := w.typeToIndex[typ]
	if !ok {
		return nil
	}
	return w.storages[id].(*Storage[T])
}

// ClearScope очищает все компоненты, зарегистрированные для указанной фазы.
// Вызывайте это в конце соответствующих этапов (Input, Logic).
func (w *World) ClearScope(scope Scope) {
	if list, ok := w.transients[scope]; ok {
		for _, s := range list {
			s.Clear()
		}
	}
}

// Внутренний метод для добавления в списки очистки
func (w *World) addToCleanupList(s AnyStorage, scope Scope) {
	// Persistent компоненты никуда не добавляем
	if scope == ScopePersistent {
		return
	}

	// Проверяем биты и добавляем storage в соответствующие списки.

	if scope&ScopeInput != 0 {
		w.transients[ScopeInput] = append(w.transients[ScopeInput], s)
	}

	if scope&ScopeLogic != 0 {
		w.transients[ScopeLogic] = append(w.transients[ScopeLogic], s)
	}

	if scope&ScopeFrame != 0 {
		w.transients[ScopeFrame] = append(w.transients[ScopeFrame], s)
	}
}

// EndFrame вызывает очистку финальной стадии.
func (w *World) EndFrame() {
	// Чистим все, что должно умереть в конце кадра (Events)
	w.ClearScope(ScopeFrame)
}
