package gateway

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
)

// NetworkCallback — абстракция над сетью (Server)
type NetworkCallback interface {
	SendToAgent(objectGuid types.ObjectGuid, msg interface{})
}

func (g *GameGateway) SetNetworkCallback(cb NetworkCallback) {
	g.network = cb

	eventbus.Subscribe(
		g.engine.Bus,
		eventbus.EventType(enums.EventChatOut),
		g.onChatOut,
	)
	logger.Log.Infof("Network callbacks setted: EventChatOut")
}

func (g *GameGateway) onChatOut(ev enums.ChatOutEvent) {

	// 1. Получаем Controller получателя
	ctrl := g.engine.Instance.GetController(ev.Receiver)
	if ctrl == nil {
		logger.Log.Warnf("[ChatDebug] Controller not found for receiver: %v. Message dropped.", ev.Receiver)
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
			SenderGuid: ev.Sender, // Или приведение к string ID
			Text:       ev.Text,
		},
	}

	g.network.SendToAgent(ev.Receiver, msg)
}
