package engine

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/eventbus"
)

const (
	SayRange  = 6
	YellRange = 15
)

type ChatSystem struct {
	Instance *Instance
	Bus      *eventbus.EventBus
}

func NewChatSystem(inst *Instance, bus *eventbus.EventBus) *ChatSystem {
	sys := &ChatSystem{Instance: inst, Bus: bus}
	eventbus.Subscribe(bus, eventbus.EventType(enums.EventChatRequest), sys.onChat)
	return sys
}

func (s *ChatSystem) onChat(ev enums.ChatRequestEvent) {
	senderPos := s.Instance.GetPosition(ev.Source)
	if senderPos == nil {
		return
	}

	radius := 0
	switch ev.Type {
	case types.ChatTypeSay, types.ChatTypeEmote:
		radius = SayRange
	case types.ChatTypeYell:
		radius = YellRange
	case types.ChatTypeWhisper:
		s.Bus.Publish(eventbus.EventType(enums.EventChatOut), enums.ChatOutEvent{
			Receiver: ev.Target,
			Sender:   ev.Source,
			Type:     ev.Type,
			Text:     ev.Message,
		})
		return
	}

	// Broadcast
	for chunkIdx, chunk := range s.Instance.Guids {
		for slotIdx, guid := range chunk {
			if guid == 0 {
				continue
			}

			ctrl := s.Instance.Controllers[chunkIdx][slotIdx]
			if ctrl == nil {
				continue
			}

			pos := s.Instance.Positions[chunkIdx][slotIdx]
			if pos == nil {
				continue
			}

			if tileDistance(senderPos.TilePos, pos.TilePos) <= radius {
				s.Bus.Publish(eventbus.EventType(enums.EventChatOut), enums.ChatOutEvent{
					Receiver: guid,
					Sender:   ev.Source,
					Type:     ev.Type,
					Text:     ev.Message,
				})
			}
		}
	}
}

func tileDistance(a, b types.TilePos) int {
	dx := int(a.X - b.X)
	if dx < 0 {
		dx = -dx
	}
	dy := int(a.Y - b.Y)
	if dy < 0 {
		dy = -dy
	}
	return dx + dy
}
