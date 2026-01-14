package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/network/connection"
	"encoding/json"
)

type ChatGateway interface {
	HandleChat(guid engine.ObjectGuid, payload api.ChatPayload)
}

type ChatHandler struct {
	gateway ChatGateway
}

func NewChatHandler(gateway ChatGateway) *ChatHandler {
	return &ChatHandler{gateway: gateway}
}

func (h *ChatHandler) Action() string { return "CHAT" }

func (h *ChatHandler) Handle(c *connection.Client, msg api.InboundMessage) {
	var p api.ChatPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return
	}
	h.gateway.HandleChat(c.Session().ObjectGuid(), p)
}
