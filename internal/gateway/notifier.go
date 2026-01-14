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
	SendToAgent(objectGuid types.ObjectGuid, msg interface{}) bool
}

func (g *GameGateway) SetNetworkCallback(cb NetworkCallback) {
	log := logger.Log.WithField("layer", "gateway")

	if cb == nil {
		log.Warn("network callback cleared")
		g.network = nil
		return
	}

	g.network = cb

	eventbus.Subscribe(
		g.engine.Bus,
		eventbus.EventType(enums.EventChatOut),
		g.onChatOut,
	)

	log.Info("network callback set: EventChatOut subscribed")
}

func (g *GameGateway) onChatOut(ev enums.ChatOutEvent) {
	log := logger.Log.WithFields(map[string]interface{}{
		"layer":    "gateway",
		"event":    "chat_out",
		"sender":   ev.Sender,
		"receiver": ev.Receiver,
		"type":     ev.Type,
	})

	// 0. Проверка сети
	if g.network == nil {
		log.Warn("network callback not set, chat dropped")
		return
	}

	// 1. Проверяем валидность получателя
	ctrl := g.engine.Instance.GetController(ev.Receiver)
	if ctrl == nil {
		log.Warn("receiver controller not found, chat dropped")
		return
	}

	// 2. Получаем имя отправителя
	senderName := "Unknown"
	if name := g.engine.Instance.GetName(ev.Sender); name != nil {
		senderName = name.Name
	} else {
		log.Debug("sender name not found, using fallback")
	}

	// 3. Формируем DTO
	msg := api.AsyncMessage{
		Type: "CHAT",
		Data: api.ChatMessage{
			Type:       uint8(ev.Type),
			SenderName: senderName,
			SenderGuid: ev.Sender,
			Text:       ev.Text,
		},
	}

	// 4. Пытаемся отправить
	if ok := g.network.SendToAgent(ev.Receiver, msg); !ok {
		log.Warn("chat delivery failed")
	}
}
