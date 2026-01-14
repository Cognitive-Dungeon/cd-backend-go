package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/server"
	"encoding/json"
)

type LoginGateway interface {
	Login(token string) (engine.ObjectGuid, error)
}

type LoginHandler struct {
	gateway LoginGateway
}

func NewLoginHandler(gateway LoginGateway) *LoginHandler {
	return &LoginHandler{gateway: gateway}
}

func (h *LoginHandler) Action() string { return "LOGIN" }

func (h *LoginHandler) Handle(c *server.Client, msg api.InboundMessage) {
	token := msg.Token

	if token == "" && len(msg.Payload) > 0 {
		var p struct {
			Token string `json:"token"`
		}
		_ = json.Unmarshal(msg.Payload, &p)
		token = p.Token
	}

	guid, err := h.gateway.Login(token)
	if err != nil {
		return
	}

	c.OnLogin(guid)
}
