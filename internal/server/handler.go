package server

import "cognitive-server/internal/api"

type MessageHandler interface {
	Handle(client *Client, msg api.InboundMessage)
}
