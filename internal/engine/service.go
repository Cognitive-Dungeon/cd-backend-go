package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions"
	"cognitive-server/internal/network"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"
	"github.com/sirupsen/logrus"
	"sort"
	"time"
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

	handlers map[domain.ActionType]handlers.HandlerFunc

	loopState LoopState
}

func NewService() *GameService {
	worlds, allEntities := buildInitialWorld()

	s := &GameService{
		Worlds:      worlds,
		Entities:    allEntities,
		GlobalTick:  0,
		Logs:        []api.LogEntry{},
		CommandChan: make(chan domain.InternalCommand, 100),
		Hub:         network.NewBroadcaster(),
		handlers:    make(map[domain.ActionType]handlers.HandlerFunc),
		loopState:   LoopStateRunning,
	}

	s.registerHandlers()
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
	s.handlers[domain.ActionMove] = handlers.WithPayload(actions.HandleMove)
	s.handlers[domain.ActionAttack] = handlers.WithPayload(actions.HandleAttack)
	s.handlers[domain.ActionTalk] = handlers.WithPayload(actions.HandleTalk)
	s.handlers[domain.ActionInteract] = handlers.WithPayload(actions.HandleInteract)
	s.handlers[domain.ActionInit] = handlers.WithEmptyPayload(actions.HandleInit)
	s.handlers[domain.ActionWait] = handlers.WithEmptyPayload(actions.HandleWait)
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
		activeActor := s.getNextActor()

		// Если никого нет (пустой мир или все мертвы), ждем и повторяем
		if activeActor == nil {
			time.Sleep(1 * time.Second)
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
			continue
		}

		// --- ХОД ИГРОКА ---
		timeout := time.After(60 * time.Second) // Тайм-аут на ход игрока
		commandProcessed := false

		for !commandProcessed {
			select {
			case cmd := <-s.CommandChan:
				// Проверяем:
				// 1. Команду прислал тот, чей сейчас ход (cmd.Token == activeActor.ID)
				// 2. ИЛИ это системная команда (INIT), которую можно слать всегда
				isTurn := cmd.Token == activeActor.ID
				isSystem := cmd.Action == domain.ActionInit

				if isTurn || isSystem {
					if isSystem {
						// Init просто возвращает стейт, не тратит ход
						s.executeCommand(cmd, activeActor)
					} else {
						// Игровое действие
						s.executeCommand(cmd, activeActor)
						commandProcessed = true
					}
				} else {
					// Если команду прислал кто-то другой не в свой ход
					// Можно отправить ошибку этому клиенту, но пока просто игнорируем
					// log.Printf("Out of turn command from %s", cmd.Token)
				}

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
}

// executeCommand выполняет хендлер и пишет логи
func (s *GameService) executeCommand(cmd domain.InternalCommand, actor *domain.Entity) {
	handler, ok := s.handlers[cmd.Action]
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

// getNextActor возвращает сущность, чей ход наступил (с наименьшим NextActionTick)
func (s *GameService) getNextActor() *domain.Entity {
	// Фильтруем кандидатов: должны иметь AI (даже игроки) и быть живыми
	candidates := make([]*domain.Entity, 0)
	for _, e := range s.Entities {
		if e.AI != nil && e.Stats != nil && !e.Stats.IsDead {
			candidates = append(candidates, e)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Сортируем: сначала те, у кого меньше тиков
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].AI.NextActionTick < candidates[j].AI.NextActionTick
	})

	return candidates[0]
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
