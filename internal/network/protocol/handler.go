package protocol

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/network/connection"
)

type CommandHandler interface {
	Action() string
	Handle(c *connection.Client, msg api.InboundMessage)
}
