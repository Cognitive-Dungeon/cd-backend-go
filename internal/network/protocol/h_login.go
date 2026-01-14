package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/network/connection"
	"cognitive-server/pkg/logger"
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type LoginGateway interface {
	HandleLogin(token string) (types.ObjectGuid, error)
}

type LoginHandler struct {
	gateway LoginGateway
}

func NewLoginHandler(gw LoginGateway) *LoginHandler {
	return &LoginHandler{gateway: gw}
}

func (h *LoginHandler) Action() string { return "LOGIN" }

func (h *LoginHandler) Handle(c *connection.Client, msg api.InboundMessage) {
	log := logger.Log.WithFields(logrus.Fields{
		"layer":  "proto",
		"action": msg.Action,
	})

	token := msg.Token

	if token == "" && len(msg.Payload) > 0 {
		var p struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal(msg.Payload, &p); err != nil {
			log.WithError(err).Warn("invalid login payload")
			return
		}
		token = p.Token
	}

	if token == "" {
		log.Warn("empty login token")
		return
	}

	guid, err := h.gateway.HandleLogin(token)
	if err != nil {
		log.WithError(err).
			Warn("login failed")
		return
	}

	log.WithField("guid", guid).
		Info("login successful")

	c.OnLogin(guid)
}
