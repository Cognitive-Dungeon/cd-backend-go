package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/engine/model"
	"cognitive-server/internal/network/connection"
	"cognitive-server/pkg/logger"
	"encoding/json"

	"github.com/sirupsen/logrus"
)

type CastGateway interface {
	HandleCast(model.ObjectGuid, api.CastPayload)
}

type CastHandler struct {
	gateway CastGateway
}

func NewCastHandler(gw CastGateway) *CastHandler {
	return &CastHandler{gateway: gw}
}

func (h *CastHandler) Action() string { return "CAST" }

func (h *CastHandler) Handle(c *connection.Client, msg api.InboundMessage) {
	var p api.CastPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		logger.Log.WithError(err).
			WithFields(logrus.Fields{
				"layer":  "proto",
				"action": msg.Action,
				"guid":   c.Session().ObjectGuid(),
			}).
			Warn("invalid payload")
		return
	}
	h.gateway.HandleCast(c.Session().ObjectGuid(), p)
}
