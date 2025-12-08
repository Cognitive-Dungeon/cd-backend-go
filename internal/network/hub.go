package network

import (
	"cognitive-server/pkg/api"
	"sync"
)

// Broadcaster занимается только рассылкой сообщений подписчикам
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan api.ServerResponse]bool
}

func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan api.ServerResponse]bool),
	}
}

// Subscribe создает канал для нового клиента
func (b *Broadcaster) Subscribe() chan api.ServerResponse {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan api.ServerResponse, 100)
	b.subscribers[ch] = true
	return ch
}

// Unsubscribe удаляет клиента
func (b *Broadcaster) Unsubscribe(ch chan api.ServerResponse) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
}

// Broadcast отправляет сообщение всем
func (b *Broadcaster) Broadcast(msg api.ServerResponse) {
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
