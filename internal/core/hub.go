package core

import (
	"cognitive-server/internal/domain"
	"sync"
)

// Broadcaster занимается только рассылкой сообщений подписчикам
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan domain.ServerResponse]bool
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan domain.ServerResponse]bool),
	}
}

// Subscribe создает канал для нового клиента
func (b *Broadcaster) Subscribe() chan domain.ServerResponse {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan domain.ServerResponse, 100)
	b.subscribers[ch] = true
	return ch
}

// Unsubscribe удаляет клиента
func (b *Broadcaster) Unsubscribe(ch chan domain.ServerResponse) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Broadcast отправляет сообщение всем
func (b *Broadcaster) Broadcast(msg domain.ServerResponse) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- msg:
		default:
			// Пропускаем медленных клиентов
		}
	}
}
