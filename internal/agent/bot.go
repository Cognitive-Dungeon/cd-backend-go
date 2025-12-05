package agent

import (
	"cognitive-server/internal/core"
	"cognitive-server/internal/domain"
	"cognitive-server/internal/systems"
	"encoding/json"
	"time"
)

// Bot представляет собой "Игрока-компьютера"
type Bot struct {
	EntityID string
	Service  *core.GameService // Ссылка на движок (вместо WebSocket соединения)
	Inbox    chan domain.ServerResponse
}

func NewBot(entityID string, service *core.GameService) *Bot {
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
			// Имитация "думания" (чтобы не было мгновенно)
			time.Sleep(200 * time.Millisecond)
			b.makeMove(event)
		}
	}
}

func (b *Bot) makeMove(state domain.ServerResponse) {
	var me *domain.Entity

	// 1. Ищем себя в списке сущностей
	for i := range state.Entities {
		if state.Entities[i].ID == b.EntityID {
			me = &state.Entities[i]
			break
		}
	}

	if me == nil || me.IsDead {
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
	allEntities := append([]domain.Entity{}, state.Entities...)
	if player != nil {
		allEntities = append(allEntities, *player)
	}

	// 4. Вызов AI
	// Передаем allEntities, чтобы бот не пытался пройти сквозь игрока
	action, target, dx, dy := systems.ComputeNPCAction(me, player, state.World, allEntities)

	// Отправляем команду
	switch action {
	case "ATTACK":
		if target != nil {
			b.sendAttack(target.ID)
		}
	case "MOVE":
		b.sendMove(dx, dy)
	default:
		b.sendWait()
	}
}

func (b *Bot) sendMove(dx, dy int) {
	payload, _ := json.Marshal(domain.DirectionPayload{Dx: dx, Dy: dy})
	cmd := domain.ClientCommand{
		Action:  "MOVE",
		Payload: payload,
		Token:   b.EntityID, // Важно: сообщаем движку, кто мы
	}
	b.Service.ProcessCommand(cmd)
}

func (b *Bot) sendAttack(targetID string) {
	payload, _ := json.Marshal(domain.EntityPayload{TargetID: targetID})

	cmd := domain.ClientCommand{
		Action:  "ATTACK",
		Payload: payload,
		Token:   b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}

func (b *Bot) sendWait() {
	cmd := domain.ClientCommand{
		Action: "WAIT",
		Token:  b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}
