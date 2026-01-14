package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/network/connection"
	"encoding/json"
)

type MoveGateway interface {
	HandleMove(engine.ObjectGuid, api.MovePayload)
}

type MoveHandler struct {
	gateway MoveGateway
}

func NewMoveHandler(gw MoveGateway) *MoveHandler {
	return &MoveHandler{gateway: gw}
}

func (h *MoveHandler) Action() string { return "MOVE" }

func (h *MoveHandler) Handle(c *connection.Client, msg api.InboundMessage) {
	var p api.MovePayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return
	}
	h.gateway.HandleMove(c.Session().ObjectGuid(), p)
}
