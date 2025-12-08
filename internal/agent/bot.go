package agent

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"encoding/json"
	"log"
)

// Bot представляет собой "Игрока-компьютера" (Headless Agent)
type Bot struct {
	EntityID string
	Service  *engine.GameService // Ссылка на движок
	Inbox    chan api.ServerResponse
}

func NewBot(entityID string, service *engine.GameService) *Bot {
	return &Bot{
		EntityID: entityID,
		Service:  service,
		// Бот регистрируется в хабе как обычный клиент (Unicast)
		Inbox: service.Hub.Register(entityID),
	}
}

// Run запускает цикл жизни бота (должен быть запущен в горутине)
func (b *Bot) Run() {
	defer b.Service.Hub.Unregister(b.EntityID)

	for event := range b.Inbox {
		// Бот реагирует только тогда, когда Арбитр сообщает: "Твой ход"
		if event.ActiveEntityID == b.EntityID {
			// Задержку убрали для производительности, сервер сам добавит паузу для визуализации
			b.makeMove(event)
		}
	}
}

func (b *Bot) makeMove(state api.ServerResponse) {
	// 1. ОПРЕДЕЛЕНИЕ РАЗМЕРОВ МИРА
	width, height := 50, 50 // Дефолт
	if state.Grid != nil {
		width, height = state.Grid.Width, state.Grid.Height
	} else if len(state.Map) > 0 {
		// Если грид не пришел, пытаемся вычислить по тайлам
		for _, t := range state.Map {
			if t.X >= width {
				width = t.X + 1
			}
			if t.Y >= height {
				height = t.Y + 1
			}
		}
	}

	// 2. СОЗДАНИЕ ЛОКАЛЬНОГО МИРА (для расчетов физики)
	localWorld := &domain.GameWorld{
		Width:       width,
		Height:      height,
		Map:         make([][]domain.Tile, height),
		SpatialHash: make(map[int][]*domain.Entity),
	}

	// Заполняем карту стенами (неизведанное = стена, чтобы не идти в пустоту)
	for y := 0; y < height; y++ {
		localWorld.Map[y] = make([]domain.Tile, width)
		for x := 0; x < width; x++ {
			localWorld.Map[y][x] = domain.Tile{X: x, Y: y, IsWall: true}
		}
	}

	// Накладываем видимые тайлы, которые прислал сервер
	for _, tv := range state.Map {
		if tv.Y < height && tv.X < width {
			localWorld.Map[tv.Y][tv.X] = domain.Tile{
				X: tv.X, Y: tv.Y, IsWall: tv.IsWall,
			}
		}
	}

	// 3. РАЗБОР СУЩНОСТЕЙ (Entities DTO -> Domain Entity)
	var me *domain.Entity
	var targetPlayer *domain.Entity

	for _, ev := range state.Entities {
		// Конвертируем View в Entity для физического движка
		ent := &domain.Entity{
			ID:    ev.ID,
			Type:  ev.Type,
			Pos:   domain.Position{X: ev.Pos.X, Y: ev.Pos.Y},
			Stats: nil,
		}

		if ev.Stats != nil {
			ent.Stats = &domain.StatsComponent{
				HP:       ev.Stats.HP,
				MaxHP:    ev.Stats.MaxHP,
				Strength: ev.Stats.Strength,
				IsDead:   ev.Stats.IsDead, // <-- Берем из Stats
			}
		} else {
			// Если статов нет (например, это предмет или декорация),
			// считаем объект "живым" препятствием (если он Solid), либо проходимым.
			// Для целей коллизии в systems.CalculateMove важно наличие Stats != nil.

			// Если это ВРАГ или NPC, но статы почему-то не пришли (баг видимости?),
			// создаем заглушку, чтобы не проходить сквозь него.
			if ev.Type == domain.EntityTypeEnemy || ev.Type == domain.EntityTypeNPC {
				ent.Stats = &domain.StatsComponent{IsDead: false}
			}
		}

		// Ищем себя
		if ev.ID == b.EntityID {
			me = ent
			me.AI = &domain.AIComponent{IsHostile: true, Personality: "Aggressive"}
		}

		// Ищем цель (Игрок)
		if ev.Type == domain.EntityTypePlayer {
			targetPlayer = ent
		}

		localWorld.AddEntity(ent)
	}

	// 4. ВАЛИДАЦИЯ
	if me == nil {
		log.Printf("[BOT %s] Self not found in state update. Skipping turn.", b.EntityID)
		b.sendWait()
		return
	}

	// Если мы мертвы - не ходим
	if me.Stats != nil && me.Stats.IsDead {
		return
	}

	// Если игрока не видно - ждем
	if targetPlayer == nil {
		b.sendWait()
		return
	}

	// 5. ВЫЗОВ СИСТЕМЫ AI
	// Теперь у нас есть мир и сущности в нужном формате
	action, target, dx, dy := systems.ComputeNPCAction(me, targetPlayer, localWorld)

	// 6. ОТПРАВКА КОМАНДЫ
	switch action {
	case domain.ActionAttack:
		if target != nil {
			b.sendAttack(target.ID)
		} else {
			b.sendWait()
		}
	case domain.ActionMove:
		b.sendMove(dx, dy)
	default:
		b.sendWait()
	}
}

// --- Хелперы отправки команд ---

func (b *Bot) sendMove(dx, dy int) {
	payload, _ := json.Marshal(api.DirectionPayload{Dx: dx, Dy: dy})
	cmd := api.ClientCommand{
		Action:  domain.ActionMove.String(), // "MOVE"
		Payload: payload,
		Token:   b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}

func (b *Bot) sendAttack(targetID string) {
	payload, _ := json.Marshal(api.EntityPayload{TargetID: targetID})
	cmd := api.ClientCommand{
		Action:  domain.ActionAttack.String(), // "ATTACK"
		Payload: payload,
		Token:   b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}

func (b *Bot) sendWait() {
	cmd := api.ClientCommand{
		Action: domain.ActionWait.String(), // "WAIT"
		Token:  b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}
