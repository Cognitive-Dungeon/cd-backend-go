package systems

import (
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
)

const (
	SayRange  = 6
	YellRange = 15
)

type ChatSystem struct {
	Instance *model.Instance
	Bus      *eventbus.EventBus
}

func NewChatSystem(inst *model.Instance, bus *eventbus.EventBus) *ChatSystem {
	sys := &ChatSystem{Instance: inst, Bus: bus}
	eventbus.Subscribe(bus, eventbus.EventType(enums.EventChatRequest), sys.onChat)
	return sys
}

func (s *ChatSystem) onChat(ev enums.ChatRequestEvent) {
	world := s.Instance.World
	senderID := ecs.EntityID(ev.Source)

	// 1. Получаем позицию отправителя напрямую из ECS
	// Используем глобальный ID компонента для скорости
	posStorage := ecs.GetStorage[components.PositionComponent](world, components.CID_Position)
	senderPos := posStorage.Get(senderID)

	if senderPos == nil {
		return // Отправителя нет в мире или он удален
	}

	var recipients []types.ObjectGuid

	// 2. Выбираем стратегию рассылки
	switch ev.Type {
	case types.ChatTypeWhisper:
		// Личное сообщение: Видит цель и сам отправитель
		if ev.Target != 0 {
			recipients = append(recipients, ev.Target)
			recipients = append(recipients, ev.Source)
		}

	case types.ChatTypeSay, types.ChatTypeEmote:
		// FindObjectsInRange уже обновлен и использует ECS внутри
		recipients = s.Instance.FindObjectsInRange(senderPos.TilePos, SayRange)

	case types.ChatTypeYell:
		recipients = s.Instance.FindObjectsInRange(senderPos.TilePos, YellRange)
	}

	// 3. Рассылка
	// Получаем хранилище контроллеров один раз перед циклом
	ctrlStorage := ecs.GetStorage[components.ControllerComponent](world, components.CID_Controller)

	for _, receiverGuid := range recipients {
		receiverID := ecs.EntityID(receiverGuid)

		// Пропускаем NPC (у них нет ControllerComponent)
		// Проверка выполняется через ECS O(1)
		if ctrlStorage.Get(receiverID) == nil {
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
