package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/server"
	"cognitive-server/pkg/logger"
)

type Dispatcher struct {
	handlers map[string]CommandHandler
}

func NewDispatcher(handlers ...CommandHandler) *Dispatcher {
	m := make(map[string]CommandHandler)
	for _, h := range handlers {
		m[h.Action()] = h
	}
	return &Dispatcher{handlers: m}
}

func (d *Dispatcher) Handle(c *server.Client, msg api.InboundMessage) {
	h, ok := d.handlers[msg.Action]
	if !ok {
		logger.Log.Warnf("Unknown action: %s", msg.Action)
		return
	}

	// Общая проверка авторизации
	if msg.Action != "LOGIN" && !c.Session().IsAuthenticated() {
		logger.Log.Warn("Ignored command before LOGIN")
		return
	}

	h.Handle(c, msg)
}
