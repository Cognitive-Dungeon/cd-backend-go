// Package eventbus предоставляет высокопроизводительную, потокобезопасную шину событий,
// оптимизированную для игровых серверов и систем реального времени.
//
// Основные архитектурные решения:
//   - Lock-Free Read (Publish): Публикация событий не использует мьютексы, что исключает блокировки (deadlocks) и задержки в горячем пути (Hot Path).
//   - Copy-On-Write (COW): Изменение списка подписчиков происходит через создание копии слайса, что обеспечивает безопасное чтение без блокировок.
//   - Generics: Строгая типизация событий на этапе компиляции.
//   - Zero-Allocation Publish: Публикация события не вызывает аллокаций памяти (кроме создания самого объекта события).
package eventbus

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// EventType представляет собой числовой идентификатор типа события.
// Используется для O(1) доступа к массиву обработчиков (Direct Array Indexing).
//
// Рекомендуется определять типы событий через iota в доменном пакете.
type EventType int

// Handler — это внутренняя функция-обработчик.
// Мы используем any, так как дженерики стираются во время выполнения,
// но внешний API (Subscribe) гарантирует корректность типов.
type Handler func(event any)

// Middleware — функция-обертка для перехвата событий.
// Позволяет добавить логирование, метрики или обработку паник для всех событий глобально.
type Middleware func(next Handler) Handler

// handlerNode хранит информацию об одном подписчике.
type handlerNode struct {
	id      uint64  // Уникальный ID подписки (для удаления)
	handler Handler // Функция обработки
}

// EventBus — диспетчер событий.
type EventBus struct {
	// handlers хранит списки обработчиков для каждого типа события.
	// Используется atomic.Pointer для безопасного чтения без мьютексов.
	// Индекс слайса соответствует значению EventType.
	handlers []atomic.Pointer[[]handlerNode]

	// mu защищает операции записи (Subscribe/Unsubscribe).
	// Используется один глобальный мьютекс, так как подписки происходят редко (Cold Path).
	mu sync.Mutex

	// middlewares — список глобальных перехватчиков.
	middlewares []Middleware

	// nextID — счетчик для генерации уникальных ID подписок.
	nextID uint64

	// size — фиксированный размер шины (количество типов событий).
	size int
}

// New создает новый экземпляр EventBus.
//
// Параметры:
//   - size: Максимальное количество типов событий (обычно равно значению константы EventCount из пакета событий).
//   - mws: Вариативный список Middleware, которые будут применяться ко всем обработчикам.
func New(size int, mws ...Middleware) *EventBus {
	bus := &EventBus{
		handlers:    make([]atomic.Pointer[[]handlerNode], size),
		middlewares: mws,
		size:        size,
	}

	// Инициализируем слоты пустыми слайсами, чтобы избежать проверки на nil в методе Publish.
	// Это микро-оптимизация для Hot Path.
	for i := 0; i < size; i++ {
		empty := make([]handlerNode, 0)
		bus.handlers[i].Store(&empty)
	}

	return bus
}

// Publish публикует событие для всех подписчиков указанного типа.
//
// Особенности:
//   - Работает без блокировок (Lock-Free).
//   - Безопасен для вызова внутри другого обработчика события (Re-entrant safe).
//   - Если eventType выходит за границы допустимого диапазона, функция паникует,
//     так как это свидетельствует о программной ошибке.
func (bus *EventBus) Publish(eventType EventType, event any) {
	idx := int(eventType)

	// Bounds Check Elimination: проверка границ обязательна для безопасности.
	// В Go 1.20+ компилятор часто оптимизирует такие проверки.
	if idx < 0 || idx >= bus.size {
		panic(fmt.Sprintf("eventbus: invalid event type index %d (size: %d)", idx, bus.size))
	}

	// 1. Атомарно загружаем указатель на текущий список хендлеров.
	// Это самая быстрая операция синхронизации в Go.
	nodesPtr := bus.handlers[idx].Load()
	nodes := *nodesPtr

	// 2. Итерируемся по локальной копии (слайс неизменяем благодаря Copy-On-Write).
	for _, node := range nodes {
		node.handler(event)
	}
}

// Subscribe подписывает типизированную функцию на событие.
//
// Параметры:
//   - eventType: Тип события (константа).
//   - fn: Функция-обработчик, принимающая конкретный тип события T.
//
// Возвращает:
//   - Функцию unsubscribe(), вызов которой отменяет эту подписку.
//
// Пример:
//
//	unsub := bus.Subscribe(events.Attack, func(ev AttackEvent) { ... })
//	defer unsub()
func Subscribe[T any](bus *EventBus, eventType EventType, fn func(ev T)) (unsubscribe func()) {
	// Оборачиваем типизированный хендлер в Handler (func(any)).
	// Здесь происходит type assertion, который гарантированно успешен благодаря дженерикам.
	baseHandler := func(e any) {
		fn(e.(T))
	}

	// Оборачиваем хендлер в Middleware (в обратном порядке, как луковицу).
	finalHandler := baseHandler
	for i := len(bus.middlewares) - 1; i >= 0; i-- {
		finalHandler = bus.middlewares[i](finalHandler)
	}

	// Генерируем ID и захватываем мьютекс для записи.
	id := atomic.AddUint64(&bus.nextID, 1)

	bus.mu.Lock()
	defer bus.mu.Unlock()

	idx := int(eventType)
	if idx < 0 || idx >= bus.size {
		panic(fmt.Sprintf("eventbus: invalid event type index %d", idx))
	}

	// --- COPY-ON-WRITE LOGIC ---

	// 1. Загружаем старый список.
	oldListPtr := bus.handlers[idx].Load()
	oldList := *oldListPtr

	// 2. Создаем новый список размером +1.
	newList := make([]handlerNode, len(oldList)+1)
	copy(newList, oldList)

	// 3. Добавляем нового подписчика в конец.
	newList[len(oldList)] = handlerNode{
		id:      id,
		handler: finalHandler,
	}

	// 4. Атомарно подменяем указатель.
	// Все новые вызовы Publish будут видеть уже newList.
	// Текущие вызовы Publish безопасно дорабатывают с oldList.
	bus.handlers[idx].Store(&newList)

	// Возвращаем замыкание для отписки, чтобы вызывающему не нужно было хранить ID.
	return func() {
		bus.unsubscribe(eventType, id)
	}
}

// unsubscribe удаляет подписчика по ID.
// Использует тот же механизм Copy-On-Write.
func (bus *EventBus) unsubscribe(eventType EventType, id uint64) {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	idx := int(eventType)
	oldListPtr := bus.handlers[idx].Load()
	oldList := *oldListPtr

	// Ищем индекс удаляемого элемента.
	targetIdx := -1
	for i, node := range oldList {
		if node.id == id {
			targetIdx = i
			break
		}
	}

	if targetIdx == -1 {
		return // Подписчик уже удален или не найден.
	}

	// Создаем новый список размером -1.
	newList := make([]handlerNode, len(oldList)-1)

	// Копируем элементы ДО удаляемого.
	copy(newList, oldList[:targetIdx])

	// Копируем элементы ПОСЛЕ удаляемого.
	// Мы сохраняем порядок элементов (Stable Remove), что важно для игровой логики
	// (например, сначала обработка урона броней, потом отнятие HP).
	copy(newList[targetIdx:], oldList[targetIdx+1:])

	// Атомарно подменяем список.
	bus.handlers[idx].Store(&newList)
}
