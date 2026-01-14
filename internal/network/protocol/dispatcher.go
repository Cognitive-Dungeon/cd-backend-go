package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/network/connection"
	"cognitive-server/pkg/logger"

	"github.com/sirupsen/logrus"
)

type Dispatcher interface {
	Handle(c *connection.Client, msg api.InboundMessage)
}

type dispatcher struct {
	handlers map[string]CommandHandler
}

func NewDispatcher(handlers ...CommandHandler) Dispatcher {
	m := make(map[string]CommandHandler, len(handlers))
	for _, h := range handlers {
		action := h.Action()
		if action == "" {
			logger.Log.
				WithField("layer", "proto").
				Error("command handler with empty action ignored")
			continue
		}

		if _, exists := m[action]; exists {
			logger.Log.
				WithFields(logrus.Fields{
					"layer":  "proto",
					"action": action,
				}).
				Error("duplicate command handler, overwriting")
		}

		m[action] = h
	}

	return &dispatcher{handlers: m}
}

func (d *dispatcher) Handle(c *connection.Client, msg api.InboundMessage) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":  "proto",
		"action": msg.Action,
	})

	// Проверка допуска до LOGIN
	if msg.Action != "LOGIN" && !c.Session().IsAuthenticated() {
		log.WithField("guid", c.Session().ObjectGuid()).
			Warn("command before login ignored")
		return
	}

	h, ok := d.handlers[msg.Action]
	if !ok {
		log.WithField("guid", c.Session().ObjectGuid()).
			Warn("unknown action")
		return
	}

	// Защита от паник внутри handler'ов
	defer func() {
		if r := recover(); r != nil {
			log.WithFields(logrus.Fields{
				"guid":  c.Session().ObjectGuid(),
				"panic": r,
			}).Error("panic in command handler")
		}
	}()

	h.Handle(c, msg)
}
