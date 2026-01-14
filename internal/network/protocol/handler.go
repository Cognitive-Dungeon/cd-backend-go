package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/server"
)

type CommandHandler interface {
	Action() string
	Handle(c *server.Client, msg api.InboundMessage)
}
