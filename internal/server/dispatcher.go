package server

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/gateway"
	"cognitive-server/pkg/logger"
	"encoding/json"
)

type Dispatcher struct {
	gateway *gateway.GameGateway
}

func NewDispatcher(gw *gateway.GameGateway) *Dispatcher {
	return &Dispatcher{gateway: gw}
}

func (d *Dispatcher) Handle(c *Client, msg api.InboundMessage) {
	logger.Log.Debugf("Protocol recv: %s", msg.Action)

	// До логина — только LOGIN
	if c.objectGuid == 0 && msg.Action != "LOGIN" {
		logger.Log.Warn("Ignored command before LOGIN")
		return
	}

	switch msg.Action {

	case "LOGIN":
		token := msg.Token

		if token == "" && len(msg.Payload) > 0 {
			var p struct {
				Token string `json:"token"`
			}
			_ = json.Unmarshal(msg.Payload, &p)
			token = p.Token
		}

		d.gateway.HandleLogin(token, func(guid engine.ObjectGuid) {
			c.objectGuid = guid
			c.server.registerClient(guid, c)
			go c.writeLoop()
		})

	case "MOVE":
		var p api.MovePayload
		if err := json.Unmarshal(msg.Payload, &p); err == nil {
			d.gateway.HandleMove(c.objectGuid, p)
		}

	case "CAST":
		var p api.CastPayload
		if err := json.Unmarshal(msg.Payload, &p); err == nil {
			d.gateway.HandleCast(c.objectGuid, p)
		}

	case "CHAT":
		var p api.ChatPayload
		if err := json.Unmarshal(msg.Payload, &p); err == nil {
			d.gateway.HandleChat(c.objectGuid, p)
		}

	default:
		logger.Log.Warnf("Unknown action: %s", msg.Action)
	}
}
