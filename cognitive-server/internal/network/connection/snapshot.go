package connection

import "cognitive-server/internal/api"

type SnapshotProvider interface {
	GetSnapshotFor(c *Client) *api.ServerResponse
}
