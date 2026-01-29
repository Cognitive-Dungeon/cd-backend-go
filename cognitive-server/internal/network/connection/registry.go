package connection

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/pkg/logger"
	"sync"

	"github.com/sirupsen/logrus"
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
	log := logger.Log.WithFields(logrus.Fields{
		"layer": "registry",
		"guid":  guid,
	})

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[guid]; exists {
		log.Warn("client already registered, overwriting")
	}

	r.clients[guid] = c
	log.Debug("client registered")
}

func (r *ClientRegistry) Unregister(guid types.ObjectGuid) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer": "registry",
		"guid":  guid,
	})

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clients[guid]; !exists {
		log.Debug("unregister called for missing client")
		return
	}

	delete(r.clients, guid)
	log.Debug("client unregistered")
}

func (r *ClientRegistry) SendToAgent(guid types.ObjectGuid, msg interface{}) bool {
	log := logger.Log.WithFields(logrus.Fields{
		"layer": "registry",
		"guid":  guid,
	})

	r.mu.RLock()
	client := r.clients[guid]
	r.mu.RUnlock()

	if client == nil {
		// Нормальная ситуация: клиент мог отключиться
		log.Debug("send dropped: client not found")
		return false
	}

	if ok := client.Send(msg); !ok {
		// Клиент жив, но не успевает читать
		log.Warn("send dropped: outgoing queue full")
	}
	return true
}
