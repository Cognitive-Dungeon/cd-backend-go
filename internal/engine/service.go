package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions"
	"cognitive-server/internal/engine/handlers/events"
	"cognitive-server/internal/network"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type LoopState uint8

const (
	LoopStateRunning LoopState = iota // Цикл активен и обрабатывает ходы.
	LoopStatePaused                   // Цикл на паузе, т.к. нет игроков.
)

type GameService struct {
	// Worlds хранит все загруженные/сгенерированные уровни
	Worlds map[int]*domain.GameWorld

	GlobalTick int

	// Entities хранит указатели на ВСЕ сущности (Игроки, NPC, Монстры)
	Entities []*domain.Entity

	Logs []api.LogEntry

	CommandChan chan domain.InternalCommand
	Hub         *network.Broadcaster

	// Канал для безопасного добавления новых игроков в работающий цикл
	JoinChan       chan *domain.Entity
	DisconnectChan chan string

	actionHandlers map[domain.ActionType]handlers.HandlerFunc
	eventHandlers  map[domain.EventType]handlers.HandlerFunc

	TurnManager *TurnManager // Менеджер очередности ходов

	loopState LoopState
}

func NewService() *GameService {
	worlds, allEntities := buildInitialWorld()

	s := &GameService{
		Worlds:         worlds,
		Entities:       allEntities,
		GlobalTick:     0,
		Logs:           []api.LogEntry{},
		CommandChan:    make(chan domain.InternalCommand, 100),
		JoinChan:       make(chan *domain.Entity, 10),
		DisconnectChan: make(chan string, 10),
		Hub:            network.NewBroadcaster(),
		actionHandlers: make(map[domain.ActionType]handlers.HandlerFunc),
		eventHandlers:  make(map[domain.EventType]handlers.HandlerFunc),
		TurnManager:    NewTurnManager(),
		loopState:      LoopStateRunning,
	}

	s.registerHandlers()

	// Add existing entities to manager
	for _, e := range s.Entities {
		if e.Stats != nil && !e.Stats.IsDead {
			s.TurnManager.AddEntity(e)
		}
	}
	// s.initTurnQueue() - removed

	return s
}

// GetEntity ищет сущность по ID во всех загруженных мирах.
func (s *GameService) GetEntity(id string) *domain.Entity {
	finderLogger := logger.Log.WithField("entity_id", id)

	finderLogger.Debug("Searching for entity...")

	for level, world := range s.Worlds {
		if entity := world.GetEntity(id); entity != nil {
			finderLogger.WithField("found_in_level", level).Debug("Entity found.")
			return entity
		}
	}

	finderLogger.Warn("Entity not found in any world.")
	return nil
}

func (s *GameService) registerHandlers() {
	s.actionHandlers[domain.ActionMove] = handlers.WithPayload(actions.HandleMove)
	s.actionHandlers[domain.ActionAttack] = handlers.WithPayload(actions.HandleAttack)
	s.actionHandlers[domain.ActionTalk] = handlers.WithPayload(actions.HandleTalk)
	s.actionHandlers[domain.ActionInteract] = handlers.WithPayload(actions.HandleInteract)
	s.actionHandlers[domain.ActionInit] = handlers.WithEmptyPayload(actions.HandleInit)
	s.actionHandlers[domain.ActionWait] = handlers.WithEmptyPayload(actions.HandleWait)

	// Inventory actions
	s.actionHandlers[domain.ActionPickup] = handlers.WithPayload(actions.HandlePickup)
	s.actionHandlers[domain.ActionDrop] = handlers.WithPayload(actions.HandleDrop)
	s.actionHandlers[domain.ActionUse] = handlers.WithPayload(actions.HandleUse)
	s.actionHandlers[domain.ActionEquip] = handlers.WithPayload(actions.HandleEquip)
	s.actionHandlers[domain.ActionUnequip] = handlers.WithPayload(actions.HandleUnequip)

	// Events
	s.eventHandlers[domain.EventLevelTransition] = handlers.WithPayload(events.HandleLevelTransition)
}

func (s *GameService) Start() {
	go s.RunGameLoop()
}

// ProcessCommand принимает команду от внешнего мира (WebSocket)
// Валидация прав доступа (Token) должна происходить ДО этого метода или внутри хендлеров,
// но здесь мы доверяем, что Token соответствует ActorID.
func (s *GameService) ProcessCommand(externalCmd api.ClientCommand) {
	actionType := domain.ParseAction(externalCmd.Action)
	if actionType == domain.ActionUnknown {
		logger.Log.WithField("action", externalCmd.Action).Warn("Unknown action received from client")
		return
	}

	s.CommandChan <- domain.InternalCommand{
		Action:  actionType,
		Token:   externalCmd.Token, // ID сущности, выполняющей действие
		Payload: externalCmd.Payload,
	}
}

// --- GAME LOOP ---

func (s *GameService) RunGameLoop() {
	logger.Log.Info("Game loop started.")

	for {
		// 1. Проверяем, есть ли новые игроки
		select {
		case newEntity := <-s.JoinChan:
			s.registerNewEntity(newEntity)
		// Очищаем "зависшие" отключения, которые произошли не в свой ход
		case <-s.DisconnectChan:
			// Просто вычитываем, чтобы канал не забился.
			// Если ход не этого игрока, нам все равно, Hub.Unregister уже сработал в main.go
		default:
			// Никого нет, продолжаем
		}

		// Если в хабе нет ни одного подписчика (ни игроков, ни ботов),
		// ставим симуляцию на паузу, чтобы не тратить ресурсы и не "прокручивать" время.
		hasSubscribers := s.Hub.SubscriberCount() > 0

		// Случай 1: Должны быть на паузе, но сейчас работаем.
		// Переходим в состояние паузы и логируем это ОДИН РАЗ.
		if !hasSubscribers && s.loopState == LoopStateRunning {
			logger.Log.Info("Game loop paused: no subscribers.")
			s.loopState = LoopStatePaused
		}

		// Случай 2: Должны работать, но сейчас на паузе.
		// Переходим в рабочее состояние и логируем это ОДИН РАЗ.
		if hasSubscribers && s.loopState == LoopStatePaused {
			logger.Log.Info("Game loop resumed.")
			s.loopState = LoopStateRunning
		}

		// Если мы на паузе, просто спим и переходим к следующей итерации.
		if s.loopState == LoopStatePaused {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 1. Кто ходит следующим?
		activeItem := s.TurnManager.PeekNext()

		// Если никого нет (пустой мир или все мертвы), ждем и повторяем
		if activeItem == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		activeActor := activeItem.Value

		if activeActor.Stats != nil && activeActor.Stats.IsDead {
			// Удаляем из очереди ходов — мертвые не ходят, кем бы они ни были
			s.TurnManager.RemoveEntity(activeActor.ID)

			// Логируем смерть. Это событие должно быть видно окружающим.
			// Тип лога COMBAT, чтобы клиент мог выделить красным.
			s.AddLog(fmt.Sprintf("%s погибает.", activeActor.Name), "COMBAT")

			// Если у сущности есть "душа" (подписчик/контроллер), принудительно обновляем его состояние.
			// Это нужно, чтобы клиент увидел HP: 0 и экран смерти, или микросервис получил сигнал о гибели.
			if s.Hub.HasSubscriber(activeActor.ID) {
				s.publishUpdate(activeActor.ID)
			}

			// Пропускаем ход, переходим к следующему
			continue
		}

		// Обновляем глобальное время
		s.GlobalTick = activeActor.AI.NextActionTick

		// 2. Рассылаем обновление всем клиентам
		// Передаем activeActor.ID, чтобы клиенты знали, чей ход (подсветка интерфейса)
		s.publishUpdate(activeActor.ID)

		// 3. Логика хода
		// Проверяем, управляется ли сущность человеком.
		// Критерий: Есть ControllerID (устанавливается при логине) ИЛИ просто есть подписчик в Hub.
		// Для надежности будем проверять наличие активного соединения в Hub.
		isHumanControlled := s.Hub.HasSubscriber(activeActor.ID)

		logger.Log.WithFields(logrus.Fields{
			"component":         "game_loop",
			"tick":              s.GlobalTick,
			"active_actor_id":   activeActor.ID,
			"active_actor_name": activeActor.Name,
			"is_human":          isHumanControlled,
		}).Debug("--- New Turn ---")

		if !isHumanControlled {
			// --- ХОД ИИ ---
			s.processAITurn(activeActor)
		} else {
			// --- ХОД ИГРОКА ---
			timeout := time.After(60 * time.Second) // Тайм-аут на ход игрока
			commandProcessed := false

			for !commandProcessed {
				select {
				// 1. ВАЖНО: Слушаем новых игроков ДАЖЕ пока ждем хода старого
				case newEntity := <-s.JoinChan:
					s.registerNewEntity(newEntity)
					// Сразу отправляем новому игроку состояние мира, не дожидаясь конца хода текущего
					if s.Hub.HasSubscriber(newEntity.ID) {
						state := s.BuildStateFor(newEntity, activeActor.ID)
						s.Hub.SendTo(newEntity.ID, *state)
					}

				case disconnectedID := <-s.DisconnectChan:
					// Если отключился тот, чей сейчас ход - прерываем ожидание!
					if disconnectedID == activeActor.ID {
						logger.Log.WithField("actor", activeActor.Name).Info("Active player disconnected. Skipping turn.")

						// Заставляем его ждать (или можно вообще ничего не делать, просто передать ход)
						activeActor.AI.Wait(domain.TimeCostWait)

						// Удаляем ControllerID, чтобы в следующем круге он считался AI (или просто стоял)
						activeActor.ControllerID = ""

						commandProcessed = true
					}

				// 2. Обработка команд
				case cmd := <-s.CommandChan:
					// Находим того, кто прислал команду (может быть не активный игрок, а тот, кто делает INIT)
					requester := s.GetEntity(cmd.Token)
					if requester == nil {
						continue
					}

					isTurn := cmd.Token == activeActor.ID
					isSystem := cmd.Action == domain.ActionInit

					if isTurn || isSystem {
						// Выполняем команду
						s.executeCommand(cmd, requester)

						// Если это INIT (реконнект или вход), принудительно шлем состояние
						if isSystem {
							if s.Hub.HasSubscriber(requester.ID) {
								state := s.BuildStateFor(requester, activeActor.ID)
								s.Hub.SendTo(requester.ID, *state)
							}
						} else {
							// Если это был игровой ход - выходим из цикла ожидания
							commandProcessed = true
						}
					}

				// 3. Тайм-аут
				case <-timeout:
					logger.Log.WithFields(logrus.Fields{
						"actor_id":   activeActor.ID,
						"actor_name": activeActor.Name,
					}).Warn("Player turn timed out. Forcing WAIT action.")
					activeActor.AI.Wait(domain.TimeCostWait)
					commandProcessed = true
				}
			}
		}

		// В конце хода обновляем приоритет в очереди
		s.TurnManager.UpdatePriority(activeActor.ID, activeActor.AI.NextActionTick)
	}
}

// registerNewEntity безопасно добавляет сущность в структуры движка
func (s *GameService) registerNewEntity(e *domain.Entity) {
	world, ok := s.Worlds[e.Level]
	if !ok {
		logger.Log.Warnf("Cannot join entity %s: level %d not found", e.ID, e.Level)
		return
	}

	// 1. Добавляем в глобальный список
	s.Entities = append(s.Entities, e)

	// 2. Добавляем в мир (Реестр + Карта)
	world.RegisterEntity(e)
	world.AddEntity(e)

	// 3. Добавляем в очередь ходов (синхронизируя время)
	if e.Stats != nil && !e.Stats.IsDead {
		// Синхронизируем время, чтобы новичок не делал 100 ходов подряд
		e.AI.NextActionTick = s.GlobalTick
		s.TurnManager.AddEntity(e)
	}

	logger.Log.WithField("id", e.ID).Info("New entity joined the game loop")
}

// executeCommand выполняет хендлер и пишет логи
func (s *GameService) executeCommand(cmd domain.InternalCommand, actor *domain.Entity) {
	handler, ok := s.actionHandlers[cmd.Action]
	if !ok {
		return
	}

	actorWorld, ok := s.Worlds[actor.Level]
	if !ok {
		logger.Log.WithFields(logrus.Fields{
			"actor_id": actor.ID,
			"level":    actor.Level,
		}).Error("executeCommand failed: Actor is on a non-existent level.")
		return
	}

	ctx := handlers.Context{
		Finder:   s,
		World:    actorWorld,
		Entities: s.Entities, // Передаем весь список
		Actor:    actor,
	}

	result, _ := handler(ctx, cmd.Payload)

	// --- Обработка события, если оно есть ---
	if result.Event != nil {
		s.processEvent(actor, result.Event)
	}

	// Логирование результата
	if result.Msg != "" {
		msgType := result.MsgType
		if msgType == "" {
			msgType = "INFO"
		}
		s.AddLog(result.Msg, msgType)
	}
}

func (s *GameService) AddLog(text, logType string) {
	s.Logs = append(s.Logs, api.LogEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:      text,
		Type:      logType,
		Timestamp: time.Now().UnixMilli(),
	})
	logger.Log.WithFields(logrus.Fields{
		"component": "game_log",
		"log_type":  logType,
	}).Info(text)
}
