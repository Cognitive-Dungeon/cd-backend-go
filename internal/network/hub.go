package network

import (
	"cognitive-server/pkg/api"
	"sync"
)

// Broadcaster занимается только рассылкой сообщений подписчикам
type Broadcaster struct {
	mu sync.RWMutex
	// Мапа: EntityID -> Личный канал
	subscribers map[string]chan api.ServerResponse
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[string]chan api.ServerResponse),
	}
}

// Register создает личный канал для сущности (Игрока или Бота)
func (b *Broadcaster) Register(entityID string) chan api.ServerResponse {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Если канал был, закрываем
	if old, ok := b.subscribers[entityID]; ok {
		close(old)
	}

	ch := make(chan api.ServerResponse, 100)
	b.subscribers[entityID] = ch
	return ch
}

// Unregister удаляет подписчика
func (b *Broadcaster) Unregister(entityID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if ch, ok := b.subscribers[entityID]; ok {
		close(ch)
		delete(b.subscribers, entityID)
	}
}

// SendTo отправляет сообщение конкретному ID (Unicast)
func (b *Broadcaster) SendTo(entityID string, msg api.ServerResponse) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if ch, ok := b.subscribers[entityID]; ok {
		select {
		case ch <- msg:
		default:
			// log.Println("Hub: Channel full for", entityID)
		}
	}
}

// Broadcast отправляет всем (нужен для зрителей/игроков)
func (b *Broadcaster) Broadcast(msg api.ServerResponse) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

// HasSubscriber проверяет, управляется ли сущность кем-то
// Используется для оптимизации (чтобы не считать AI для тех, кого нет)
func (b *Broadcaster) HasSubscriber(entityID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.subscribers[entityID]
	return ok
}
