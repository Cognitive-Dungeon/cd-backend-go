package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/server"
	"encoding/json"
)

type CastGateway interface {
	HandleCast(engine.ObjectGuid, api.CastPayload)
}

type CastHandler struct {
	gateway CastGateway
}

func NewCastHandler(gw CastGateway) *CastHandler {
	return &CastHandler{gateway: gw}
}

func (h *CastHandler) Action() string { return "CAST" }

func (h *CastHandler) Handle(c *server.Client, msg api.InboundMessage) {
	var p api.CastPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return
	}
	h.gateway.HandleCast(c.Session().ObjectGuid(), p)
}
