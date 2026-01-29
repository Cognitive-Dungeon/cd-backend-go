package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/network/connection"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
)

type ChatGateway interface {
	HandleChat(guid types.ObjectGuid, payload api.ChatPayload)
}

type ChatHandler struct {
	gateway ChatGateway
}

func NewChatHandler(gw ChatGateway) *ChatHandler {
	return &ChatHandler{gateway: gw}
}

func (h *ChatHandler) Action() string { return "CHAT" }

func (h *ChatHandler) Handle(c *connection.Client, msg api.InboundMessage) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":  "proto",
		"action": msg.Action,
		"guid":   c.Session().ObjectGuid(),
	})

	var p api.ChatPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		log.WithError(err).Warn("invalid CHAT payload")
		return
	}

	p.Message = strings.TrimSpace(p.Message)
	if p.Message == "" {
		log.Debug("empty chat message ignored")
		return
	}

	h.gateway.HandleChat(c.Session().ObjectGuid(), p)
}
