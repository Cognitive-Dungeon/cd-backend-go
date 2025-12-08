package agent

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"encoding/json"
	"log"
)

// Bot представляет собой "Игрока-компьютера" (Headless Agent).
// Этот код является примером ВНЕШНЕГО клиента, который подключается к серверу
// так же, как и обычный игрок через WebSocket. Он получает обновления мира
// и на их основе принимает решение, какую команду отправить обратно.
//
// Жизненный цикл:
//  1. NewBot -> Регистрация в хабе сервера, получение личного канала (Inbox).
//  2. Run -> Запуск в отдельной горутине, слушает свой Inbox.
//  3. При получении события, если сейчас ход бота (ActiveEntityID == EntityID),
//     вызывается makeMove.
//  4. makeMove -> Анализирует полученное состояние мира, вызывает систему AI и отправляет команду.
type Bot struct {
	EntityID string
	Service  *engine.GameService // Прямая ссылка на движок (для простоты в этом проекте)
	Inbox    chan api.ServerResponse
}

func NewBot(entityID string, service *engine.GameService) *Bot {
	log.Printf("[BOT] Creating agent for entity %s", entityID)
	return &Bot{
		EntityID: entityID,
		Service:  service,
		// Бот регистрируется в хабе как обычный клиент и получает свой канал для обновлений.
		Inbox: service.Hub.Register(entityID),
	}
}

// Run запускает цикл жизни бота. Должен быть запущен в горутине.
func (b *Bot) Run() {
	defer b.Service.Hub.Unregister(b.EntityID)

	for event := range b.Inbox {
		// Бот реагирует только тогда, когда Арбитр сообщает: "Твой ход".
		if event.ActiveEntityID == b.EntityID {
			b.makeMove(event)
		}
	}
	log.Printf("[BOT] Agent for %s shut down.", b.EntityID)
}

// makeMove — это мозг бота. Он принимает решение на основе полученного состояния мира.
func (b *Bot) makeMove(state api.ServerResponse) {
	// --- ШАГ 1: ВОССОЗДАНИЕ ЛОКАЛЬНОЙ КАРТИНЫ МИРА ---
	// Бот, как и игрок, получает только ЧАСТЬ информации о мире (то, что видит).
	// Чтобы его внутренние системы (например, поиск пути) могли работать,
	// он должен построить свою локальную копию мира на основе этих данных.
	localWorld, err := b.buildLocalWorld(state)
	if err != nil {
		log.Printf("[BOT %s] Error building local world: %v. Waiting.", b.EntityID, err)
		b.sendWait()
		return
	}

	// --- ШАГ 2: ПРЕОБРАЗОВАНИЕ DTO В ДОМЕННЫЕ СУЩНОСТИ ---
	// Сервер присылает данные в формате DTO (Data Transfer Object) из пакета api.
	// Системы ИИ работают с доменными сущностями (domain.Entity).
	// Нам нужно найти себя и потенциальную цель в полученных данных.
	me, targetPlayer := b.findActors(state, localWorld)

	// --- ШАГ 3: ВАЛИДАЦИЯ СОСТОЯНИЯ ---
	if me == nil {
		log.Printf("[BOT %s] Self not found in state update. Skipping turn.", b.EntityID)
		b.sendWait()
		return
	}
	if me.Stats != nil && me.Stats.IsDead {
		return // Мертвые не ходят
	}
	if targetPlayer == nil {
		// Если игрока не видно - просто ждем.
		// В будущем здесь может быть логика патрулирования.
		b.sendWait()
		return
	}

	// --- ШАГ 4: ВЫЗОВ СИСТЕМЫ ПРИНЯТИЯ РЕШЕНИЙ (AI) ---
	// Теперь, когда у нас есть локальный мир и акторы, мы можем использовать
	// ту же самую систему AI, что и сервер, для вычисления следующего действия.
	action, target, dx, dy := systems.ComputeNPCAction(me, targetPlayer, localWorld)

	// --- ШАГ 5: ОТПРАВКА КОМАНДЫ НА СЕРВЕР ---
	switch action {
	case domain.ActionAttack:
		if target != nil {
			b.sendAttack(target.ID)
		} else {
			b.sendWait() // Не удалось найти цель для атаки
		}
	case domain.ActionMove:
		b.sendMove(dx, dy)
	default:
		b.sendWait()
	}
}

// buildLocalWorld создает локальную копию мира из данных, полученных от сервера.
func (b *Bot) buildLocalWorld(state api.ServerResponse) (*domain.GameWorld, error) {
	width, height := 50, 50 // Дефолт, если грид не пришел
	if state.Grid != nil {
		width, height = state.Grid.Width, state.Grid.Height
	}

	localWorld := &domain.GameWorld{
		Width:       width,
		Height:      height,
		Map:         make([][]domain.Tile, height),
		SpatialHash: make(map[int][]*domain.Entity),
	}

	// Заполняем карту непроходимыми стенами по умолчанию.
	// Бот считает всё, что не видел, стеной, чтобы не строить пути в неизвестность.
	for y := 0; y < height; y++ {
		localWorld.Map[y] = make([]domain.Tile, width)
		for x := 0; x < width; x++ {
			localWorld.Map[y][x] = domain.Tile{X: x, Y: y, IsWall: true}
		}
	}

	// Накладываем видимые тайлы, которые прислал сервер.
	for _, tv := range state.Map {
		if tv.Y < height && tv.X < width {
			localWorld.Map[tv.Y][tv.X] = domain.Tile{X: tv.X, Y: tv.Y, IsWall: tv.IsWall}
		}
	}
	return localWorld, nil
}

// findActors ищет в DTO-ответе сервера себя и цель, конвертирует их в domain.Entity
// и добавляет в локальный мир.
func (b *Bot) findActors(state api.ServerResponse, localWorld *domain.GameWorld) (me, target *domain.Entity) {
	for _, ev := range state.Entities {
		// Конвертируем EntityView (DTO) в domain.Entity для физического движка
		ent := &domain.Entity{
			ID:    ev.ID,
			Type:  ev.Type,
			Pos:   domain.Position{X: ev.Pos.X, Y: ev.Pos.Y},
			Stats: &domain.StatsComponent{IsDead: true}, // По умолчанию считаем мертвым/непроходимым
		}

		if ev.Stats != nil {
			ent.Stats = &domain.StatsComponent{
				HP:       ev.Stats.HP,
				MaxHP:    ev.Stats.MaxHP,
				Strength: ev.Stats.Strength,
				IsDead:   ev.Stats.IsDead,
			}
		}

		// Ищем себя и цель
		if ev.ID == b.EntityID {
			me = ent
			me.AI = &domain.AIComponent{IsHostile: true} // Предполагаем, что бот всегда враждебен
		}
		if ev.Type == domain.EntityTypePlayer {
			target = ent
		}

		localWorld.AddEntity(ent)
	}
	return me, target
}

// --- Хелперы для отправки команд на сервер ---

func (b *Bot) sendCommand(action domain.ActionType, payload interface{}) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[BOT %s] Error marshalling payload: %v", b.EntityID, err)
		return
	}

	cmd := api.ClientCommand{
		Action:  action.String(),
		Payload: payloadBytes,
		Token:   b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}

func (b *Bot) sendMove(dx, dy int) {
	b.sendCommand(domain.ActionMove, api.DirectionPayload{Dx: dx, Dy: dy})
}

func (b *Bot) sendAttack(targetID string) {
	b.sendCommand(domain.ActionAttack, api.EntityPayload{TargetID: targetID})
}

func (b *Bot) sendWait() {
	cmd := api.ClientCommand{
		Action: domain.ActionWait.String(),
		Token:  b.EntityID,
	}
	b.Service.ProcessCommand(cmd)
}
