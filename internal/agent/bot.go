package agent

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"encoding/json"
)

// Bot представляет собой "Игрока-компьютера"
type Bot struct {
	EntityID string
	Service  *engine.GameService // Ссылка на движок (вместо WebSocket соединения)
	Inbox    chan api.ServerResponse
}

func NewBot(entityID string, service *engine.GameService) *Bot {
	return &Bot{
		EntityID: entityID,
		Service:  service,
		Inbox:    service.Hub.Subscribe(), // Бот подписывается как обычный игрок
	}
}

// Run запускает цикл жизни бота
func (b *Bot) Run() {
	defer b.Service.Hub.Unsubscribe(b.Inbox)

	for event := range b.Inbox {
		// Мы реагируем только если сервер сказал "Твой ход" (или пришел UPDATE с нашим ID)
		// В текущей реализации мы смотрим на поле ActiveEntityID
		if event.ActiveEntityID == b.EntityID {
			b.makeMove(event)
		}
	}
}

func (b *Bot) makeMove(state api.ServerResponse) {
	var me *domain.Entity

	// 1. Ищем себя в списке сущностей
	for i := range state.Entities {
		if state.Entities[i].ID == b.EntityID {
			me = &state.Entities[i]
			break
		}
	}

	if me == nil || me.Stats == nil || me.Stats.IsDead {
		return
	}

	// 2. Игрок передается в отдельном поле ServerResponse
	player := state.Player

	if player == nil {
		b.sendWait()
		return
	}

	// 3. Важный момент: для корректного расчета пути (AI) нужно,
	// чтобы Игрок тоже считался препятствием.
	// Создаем временный слайс для AI, куда добавляем игрока.
	allEntities := make([]*domain.Entity, 0, len(state.Entities)+1)

	// Добавляем игрока
	allEntities = append(allEntities, player)

	// Добавляем NPC (берем адреса)
	for i := range state.Entities {
		allEntities = append(allEntities, &state.Entities[i])
	}

	// 4. Вызов AI
	// Передаем allEntities, чтобы бот не пытался пройти сквозь игрока
	action, target, dx, dy := systems.ComputeNPCAction(me, player, state.World, allEntities)

	// Отправляем команду
	switch action {
	case domain.ActionAttack:
		if target != nil {
			b.sendAttack(target.ID)
		}
	case domain.ActionMove:
		b.sendMove(dx, dy)
	default:
		b.sendWait()
	}
}

func (b *Bot) sendMove(dx, dy int) {
	payload, _ := json.Marshal(api.DirectionPayload{Dx: dx, Dy: dy})
	cmd := api.ClientCommand{
		Action:  domain.ActionMove.String(),
		Payload: payload,
		Token:   b.EntityID, // Важно: сообщаем движку, кто мы
	}
	b.Service.ProcessCommand(cmd)
}

func (b *Bot) sendAttack(targetID string) {
	payload, _ := json.Marshal(api.EntityPayload{TargetID: targetID})

	cmd := api.ClientCommand{
		Action:  domain.ActionAttack.String(),
		Payload: payload,
		Token:   b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}

func (b *Bot) sendWait() {
	cmd := api.ClientCommand{
		Action: domain.ActionWait.String(),
		Token:  b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}
