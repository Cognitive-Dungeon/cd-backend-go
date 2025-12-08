package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions" // Импорт конкретных реализаций
	"cognitive-server/internal/network"                 // Хаб теперь здесь
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/dungeon"
	"fmt"
	"log"
	"sort"
	"time"
)

type GameService struct {
	World    *domain.GameWorld
	Player   *domain.Entity
	Entities []domain.Entity // Храним по значению
	Logs     []api.LogEntry

	CommandChan chan domain.InternalCommand
	Hub         *network.Broadcaster

	// Реестр обработчиков: ActionType -> Функция
	handlers map[domain.ActionType]handlers.HandlerFunc
}

func NewService() *GameService {
	// Генерация мира
	world, entities, startPos := dungeon.Generate(1)

	// Создание игрока (ECS style)
	player := &domain.Entity{
		ID:   "p1",
		Name: "Герой",
		Type: domain.EntityTypePlayer,
		Pos:  startPos,

		Render: &domain.RenderComponent{Symbol: "@", Color: "text-cyan-400"},
		Stats: &domain.StatsComponent{
			HP: 100, MaxHP: 100, Stamina: 100, MaxStamina: 100, Gold: 50, Strength: 10,
		},
		AI: &domain.AIComponent{
			NextActionTick: 0,
			IsHostile:      false,
		},
		Narrative: &domain.NarrativeComponent{
			Description: "Искатель приключений с горящими глазами.",
		},
	}

	s := &GameService{
		World:       world,
		Player:      player,
		Entities:    entities,
		Logs:        []api.LogEntry{},
		CommandChan: make(chan domain.InternalCommand, 100),
		Hub:         network.NewBroadcaster(),
		handlers:    make(map[domain.ActionType]handlers.HandlerFunc),
	}

	s.registerHandlers()
	return s
}

// registerHandlers связывает Enum действий с функциями-хендлерами
func (s *GameService) registerHandlers() {
	// Команды с данными
	s.handlers[domain.ActionMove] = handlers.WithPayload(actions.HandleMove)
	s.handlers[domain.ActionAttack] = handlers.WithPayload(actions.HandleAttack)
	s.handlers[domain.ActionTalk] = handlers.WithPayload(actions.HandleTalk)

	// Команды без данных
	s.handlers[domain.ActionInit] = handlers.WithEmptyPayload(actions.HandleInit)
	s.handlers[domain.ActionWait] = handlers.WithEmptyPayload(actions.HandleWait)
}

func (s *GameService) Start() {
	go s.RunGameLoop()
}

// ProcessCommand конвертирует внешний JSON во внутреннюю команду
func (s *GameService) ProcessCommand(externalCmd api.ClientCommand) {
	actionType := domain.ParseAction(externalCmd.Action)

	if actionType == domain.ActionUnknown {
		log.Printf("[WARN] Unknown action received: %s", externalCmd.Action)
		return
	}

	internalCmd := domain.InternalCommand{
		Action:  actionType,
		Token:   externalCmd.Token,
		Payload: externalCmd.Payload,
	}

	select {
	case s.CommandChan <- internalCmd:
	default:
		log.Println("[WARN] Command queue full")
	}
}

// executeCommand находит нужный хендлер и запускает его
func (s *GameService) executeCommand(cmd domain.InternalCommand, actor *domain.Entity) {
	handler, ok := s.handlers[cmd.Action]
	if !ok {
		log.Printf("[ERROR] No handler registered for action type: %v", cmd.Action)
		return
	}

	// 1. Собираем единый список всех сущностей (Игрок + NPC)
	// Это решает проблему "Где искать цель?"
	allEntities := make([]*domain.Entity, 0, len(s.Entities)+1)
	allEntities = append(allEntities, s.Player)
	for i := range s.Entities {
		allEntities = append(allEntities, &s.Entities[i])
	}

	// 2. Создаем контекст
	ctx := handlers.Context{
		World:    s.World,
		Entities: allEntities, // Передаем список указателей
		Actor:    actor,       // Кто совершает действие
	}

	// 3. Вызываем хендлер
	result, err := handler(ctx, cmd.Payload)

	if err != nil {
		log.Printf("[ERROR] Logic error in %v: %v", cmd.Action, err)
		return
	}

	// 4. Логируем результат, если есть сообщение
	if result.Msg != "" {
		// Дефолтный тип лога INFO, если хендлер не указал иной
		msgType := result.MsgType
		if msgType == "" {
			msgType = "INFO"
		}
		s.AddLog(result.Msg, msgType)
	}
}

// --- ГЛАВНЫЙ ЦИКЛ (ARBITER LOOP) ---

func (s *GameService) RunGameLoop() {
	log.Println("[LOOP] Arbiter Loop started")

	for {
		// 1. Определяем, чей ход
		activeActor := s.getNextActor()
		s.World.GlobalTick = activeActor.AI.NextActionTick

		// 2. Уведомляем клиентов
		s.publishTurn(activeActor.ID)

		// 3. Ждем команду
		timeout := time.After(5 * time.Second)
		commandProcessed := false

		for !commandProcessed {
			select {
			case cmd := <-s.CommandChan:
				senderID := cmd.Token
				// Хак для MVP: пустой токен = Игрок
				if senderID == "" {
					senderID = s.Player.ID
				}

				// Разрешаем команду, если это ход актера ИЛИ это системная команда INIT
				isTurn := senderID == activeActor.ID
				isSystem := cmd.Action == domain.ActionInit

				if isTurn || isSystem {
					if isSystem {
						// INIT всегда выполняется от имени игрока (или системно)
						s.executeCommand(cmd, s.Player)
					} else {
						s.executeCommand(cmd, activeActor)
						commandProcessed = true // Ход сделан, выходим из ожидания
					}
				} else {
					log.Printf("[ARBITER] Ignored command from %s (current turn: %s)", senderID, activeActor.Name)
				}

			case <-timeout:
				log.Printf("[ARBITER] Timeout for %s. Forcing WAIT.", activeActor.Name)
				activeActor.AI.Wait(domain.TimeCostWait)
				commandProcessed = true
			}
		}

		// 4. Рассылаем состояние после хода
		s.publishUpdate()
	}
}

// --- УТИЛИТЫ ---

func (s *GameService) getNextActor() *domain.Entity {
	// Собираем кандидатов на ход (только живые и с AI)
	var activeEntities []*domain.Entity

	// Игрок
	if s.Player.AI != nil && s.Player.Stats != nil && !s.Player.Stats.IsDead {
		activeEntities = append(activeEntities, s.Player)
	}

	// NPC
	for i := range s.Entities {
		e := &s.Entities[i]
		if e.AI != nil && e.Stats != nil && !e.Stats.IsDead {
			activeEntities = append(activeEntities, e)
		}
	}

	// Сортировка (Priority Queue)
	sort.Slice(activeEntities, func(i, j int) bool {
		return activeEntities[i].AI.NextActionTick < activeEntities[j].AI.NextActionTick
	})

	if len(activeEntities) == 0 {
		// Критическая ситуация: все умерли или нет AI. Возвращаем игрока, чтобы цикл не падал.
		return s.Player
	}

	return activeEntities[0]
}

func (s *GameService) publishTurn(activeID string) {
	currentLogs := make([]api.LogEntry, len(s.Logs))
	copy(currentLogs, s.Logs)
	s.Logs = []api.LogEntry{}

	response := api.ServerResponse{
		Type:           "UPDATE",
		World:          s.World,
		Player:         s.Player,
		Entities:       s.Entities,
		Logs:           currentLogs,
		ActiveEntityID: activeID,
	}
	s.Hub.Broadcast(response)
}

func (s *GameService) publishUpdate() {
	// То же самое, но ActiveEntityID может быть пустым или старым
	// Для простоты используем ту же логику сборки пакета
	// Можно передавать "" в publishTurn, но лучше иметь явный метод
	s.publishTurn("")
}

func (s *GameService) GetState() *api.ServerResponse {
	currentLogs := make([]api.LogEntry, len(s.Logs))
	copy(currentLogs, s.Logs)
	s.Logs = []api.LogEntry{}

	return &api.ServerResponse{
		Type:     "UPDATE",
		World:    s.World,
		Player:   s.Player,
		Entities: s.Entities,
		Logs:     currentLogs,
	}
}

func (s *GameService) AddLog(text, logType string) {
	s.Logs = append(s.Logs, api.LogEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:      text,
		Type:      logType,
		Timestamp: time.Now().UnixMilli(),
	})
}
