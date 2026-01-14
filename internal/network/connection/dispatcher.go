package connection

import "cognitive-server/internal/api"

type MessageSink interface {
	HandleMessage(c *Client, msg api.InboundMessage)
}
