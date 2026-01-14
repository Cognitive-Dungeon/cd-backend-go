package connection

import "cognitive-server/internal/network/session"

type ClientEvents interface {
	OnAuthenticated(c *Client, s *session.Session)
	OnDisconnected(c *Client)
}
