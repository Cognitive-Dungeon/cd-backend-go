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
	// 1. Получаем позицию отправителя
	senderPos := s.Instance.GetPosition(ev.Source)
	if senderPos == nil {
		return
	} // Отправителя нет в мире

	var recipients []ObjectGuid

	// 2. Выбираем стратегию рассылки
	switch ev.Type {
	case types.ChatTypeWhisper:
		// Личное сообщение: Видит цель и сам отправитель
		if ev.Target != 0 {
			recipients = append(recipients, ev.Target)
			recipients = append(recipients, ev.Source)
		}

	case types.ChatTypeSay, types.ChatTypeEmote:
		recipients = s.Instance.FindObjectsInRange(senderPos.TilePos, SayRange)

	case types.ChatTypeYell:
		recipients = s.Instance.FindObjectsInRange(senderPos.TilePos, YellRange)
	}

	// 3. Рассылка
	for _, receiverGuid := range recipients {
		// Пропускаем NPC (у них нет контроллера, значит некому слать JSON)
		// Если в будущем NPC научатся читать чат (LLM), уберем эту проверку.
		if s.Instance.GetController(receiverGuid) == nil {
			continue
		}

		s.Bus.Publish(eventbus.EventType(enums.EventChatOut), enums.ChatOutEvent{
			Receiver: receiverGuid,
			Sender:   ev.Source,
			Type:     ev.Type,
			Text:     ev.Message,
		})
	}
}
