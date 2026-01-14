package server

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/pkg/logger"
	"sync"
)

type ClientRegistry struct {
	mu      sync.RWMutex
	clients map[types.ObjectGuid]*Client
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients: make(map[types.ObjectGuid]*Client),
	}
}

func (r *ClientRegistry) Register(guid types.ObjectGuid, c *Client) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.clients[guid] = c
}

func (r *ClientRegistry) Unregister(guid types.ObjectGuid) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.clients, guid)
}

func (r *ClientRegistry) SendToAgent(guid types.ObjectGuid, msg interface{}) {
	r.mu.RLock()
	client := r.clients[guid]
	r.mu.RUnlock()

	if client == nil {
		logger.Log.Warnf("[Network] Client not found for %s", guid)
		return
	}

	select {
	case client.sendChan <- msg:
	default:
		logger.Log.Warnf("Outgoing queue full for %s. Message dropped!", guid)
	}
}
