package gateway

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/eventbus"
)

// NetworkCallback — абстракция над сетью (Server)
type NetworkCallback interface {
	SendToAgent(agentID string, msg interface{})
}

func (g *GameGateway) SetNetworkCallback(cb NetworkCallback) {
	g.network = cb

	eventbus.Subscribe(
		g.engine.Bus,
		eventbus.EventType(enums.EventChatOut),
		g.onChatOut,
	)
}

func (g *GameGateway) onChatOut(ev enums.ChatOutEvent) {
	// 1. Получаем Controller получателя
	ctrl := g.engine.Instance.GetController(ev.Receiver)
	if ctrl == nil {
		return
	}

	// 2. Получаем имя отправителя
	senderName := "Unknown"
	if name := g.engine.Instance.GetName(ev.Sender); name != nil {
		senderName = name.Name
	}

	// 3. Формируем DTO
	msg := api.AsyncMessage{
		Type: "CHAT",
		Data: api.ChatMessage{
			Type:       uint8(ev.Type),
			SenderName: senderName,
			SenderGuid: ev.Sender.String(),
			Text:       ev.Text,
		},
	}

	// 4. Отправляем через network.Service
	g.network.SendToAgent(ctrl.AgentID, msg)
}
