package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions"
	"cognitive-server/internal/network"
	"cognitive-server/internal/systems"
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
	Entities []domain.Entity
	Logs     []api.LogEntry

	CommandChan chan domain.InternalCommand
	Hub         *network.Broadcaster

	handlers map[domain.ActionType]handlers.HandlerFunc
}

func NewService() *GameService {
	world, entities, startPos := dungeon.Generate(1)
	player := &domain.Entity{
		ID:   "p1",
		Name: "Герой",
		Type: domain.EntityTypePlayer,
		Pos:  startPos,

		Render: &domain.RenderComponent{Symbol: "@", Color: "#22D3EE"},
		Stats: &domain.StatsComponent{
			HP: 100, MaxHP: 100, Stamina: 100, MaxStamina: 100, Gold: 50, Strength: 10,
		},
		AI:        &domain.AIComponent{NextActionTick: 0, IsHostile: false},
		Narrative: &domain.NarrativeComponent{Description: "Искатель."},
		Vision:    &domain.VisionComponent{Radius: domain.VisionRadius},
		Memory:    &domain.MemoryComponent{ExploredIDs: make(map[int]bool)},
	}

	// Инициализация индексов
	world.SpatialHash = make(map[int][]*domain.Entity)
	world.EntityRegistry = make(map[string]*domain.Entity)

	world.AddEntity(player)
	world.RegisterEntity(player)
	for i := range entities {
		world.AddEntity(&entities[i])
		world.RegisterEntity(&entities[i])
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

func (s *GameService) registerHandlers() {
	s.handlers[domain.ActionMove] = handlers.WithPayload(actions.HandleMove)
	s.handlers[domain.ActionAttack] = handlers.WithPayload(actions.HandleAttack)
	s.handlers[domain.ActionTalk] = handlers.WithPayload(actions.HandleTalk)
	s.handlers[domain.ActionInit] = handlers.WithEmptyPayload(actions.HandleInit)
	s.handlers[domain.ActionWait] = handlers.WithEmptyPayload(actions.HandleWait)
}

func (s *GameService) Start() {
	go s.RunGameLoop()
}

func (s *GameService) ProcessCommand(externalCmd api.ClientCommand) {
	actionType := domain.ParseAction(externalCmd.Action)
	if actionType == domain.ActionUnknown {
		return
	}

	s.CommandChan <- domain.InternalCommand{
		Action:  actionType,
		Token:   externalCmd.Token,
		Payload: externalCmd.Payload,
	}
}

// --- GAME LOOP ---

func (s *GameService) RunGameLoop() {
	log.Println("[LOOP] Arbiter Loop started")
	for {
		activeActor := s.getNextActor()
		s.World.GlobalTick = activeActor.AI.NextActionTick

		// Уведомляем всех об актуальном состоянии
		s.publishUpdate(activeActor.ID)

		timeout := time.After(5 * time.Second)
		commandProcessed := false

		for !commandProcessed {
			select {
			case cmd := <-s.CommandChan:
				senderID := cmd.Token
				if senderID == "" {
					senderID = s.Player.ID
				} // Fallback for player

				isTurn := senderID == activeActor.ID
				isSystem := cmd.Action == domain.ActionInit

				if isTurn || isSystem {
					if isSystem {
						s.executeCommand(cmd, s.Player)
					} else {
						s.executeCommand(cmd, activeActor)
						commandProcessed = true
					}
				} else {
					// Игнор команд вне очереди
				}

			case <-timeout:
				log.Printf("[ARBITER] Timeout for %s (%s). Forcing WAIT.", activeActor.Name, activeActor.ID)
				activeActor.AI.Wait(domain.TimeCostWait)
				commandProcessed = true
			}
		}

		s.publishUpdate(activeActor.ID)

	}
}

// publishUpdate: Генерирует персональный стейт для каждого агента
func (s *GameService) publishUpdate(activeID string) {
	// Список всех получателей (Игрок + NPC)
	receivers := make([]*domain.Entity, 0, len(s.Entities)+1)
	receivers = append(receivers, s.Player)
	for i := range s.Entities {
		receivers = append(receivers, &s.Entities[i])
	}

	// Рассылка
	for _, e := range receivers {
		if s.Hub.HasSubscriber(e.ID) {
			state := s.BuildStateFor(e, activeID)
			s.Hub.SendTo(e.ID, *state)
		}
	}

	// Очистка логов ПОСЛЕ рассылки
	s.Logs = []api.LogEntry{}
}

// BuildStateFor: Создает DTO с картой тайлов (TileView) вместо сырого World
func (s *GameService) BuildStateFor(observer *domain.Entity, activeID string) *api.ServerResponse {
	// 1. FOV и Memory (без изменений)
	var visibleIdxs map[int]bool
	isGod := false
	if observer.Vision != nil {
		visibleIdxs = systems.ComputeVisibleTiles(s.World, observer.Pos, observer.Vision)
		if visibleIdxs == nil {
			isGod = true
		}
	}
	if observer.Memory != nil && !isGod {
		for idx := range visibleIdxs {
			observer.Memory.ExploredIDs[idx] = true
		}
	}

	// 2. Map DTO (без изменений)
	var mapDTO []api.TileView
	for y := 0; y < s.World.Height; y++ {
		for x := 0; x < s.World.Width; x++ {
			idx := s.World.GetIndex(x, y)
			tile := s.World.Map[y][x]
			isExplored := isGod
			if !isGod && observer.Memory != nil {
				isExplored = observer.Memory.ExploredIDs[idx]
			}
			if isExplored {
				tView := api.TileView{
					X: x, Y: y, IsWall: tile.IsWall,
					IsVisible:  isGod || visibleIdxs[idx],
					IsExplored: true,
					Symbol:     ".", Color: "#333333",
				}
				if tile.IsWall {
					tView.Symbol = "#"
					tView.Color = "#666666"
				}
				mapDTO = append(mapDTO, tView)
			}
		}
	}

	// 3. Entities DTO (Единый список!)
	var viewEntities []api.EntityView

	// Собираем всех потенциальных кандидатов (Игрок + NPC)
	// В нормальном ECS игрок лежал бы внутри Entities, но пока объединяем вручную
	allCandidates := make([]*domain.Entity, 0, len(s.Entities)+1)
	allCandidates = append(allCandidates, s.Player)
	for i := range s.Entities {
		allCandidates = append(allCandidates, &s.Entities[i])
	}

	for _, e := range allCandidates {
		// Видим ли мы его?
		idx := s.World.GetIndex(e.Pos.X, e.Pos.Y)
		isVisible := isGod || visibleIdxs[idx]

		// Самого себя видим всегда
		if e.ID == observer.ID {
			isVisible = true
		}

		if isVisible {
			// Конвертируем с учетом контекста (кто смотрит?)
			viewEntities = append(viewEntities, s.toEntityView(e, observer))
		}
	}

	// Копия логов
	logsCopy := make([]api.LogEntry, len(s.Logs))
	copy(logsCopy, s.Logs)

	return &api.ServerResponse{
		Type:           "UPDATE",
		Tick:           s.World.GlobalTick,
		MyEntityID:     observer.ID, // Говорим клиенту, кто он
		ActiveEntityID: activeID,
		Grid:           &api.GridMeta{Width: s.World.Width, Height: s.World.Height},
		Map:            mapDTO,
		Entities:       viewEntities, // <-- Все здесь
		Logs:           logsCopy,
	}
}

// Умный маппер
func (s *GameService) toEntityView(target *domain.Entity, observer *domain.Entity) api.EntityView {
	view := api.EntityView{
		ID:   target.ID,
		Type: target.Type,
		Name: target.Name,
	}
	view.Pos.X = target.Pos.X
	view.Pos.Y = target.Pos.Y

	if target.Render != nil {
		view.Render.Symbol = target.Render.Symbol
		view.Render.Color = target.Render.Color
	} else {
		view.Render.Symbol = "?"
		view.Render.Color = "#fff"
	}

	// Логика видимости статов (Security)
	// Статы показываем, если:
	// 1. Это мы сами
	// 2. Это труп (видим IsDead)
	// 3. (В будущем) Если мы скастовали "Оценку"

	isMe := target.ID == observer.ID
	isDead := target.Stats != nil && target.Stats.IsDead

	if target.Stats != nil {
		if isMe {
			// Полные статы
			view.Stats = &api.StatsView{
				HP: target.Stats.HP, MaxHP: target.Stats.MaxHP,
				Stamina: target.Stats.Stamina, MaxStamina: target.Stats.MaxStamina,
				Gold: target.Stats.Gold, Strength: target.Stats.Strength,
				IsDead: target.Stats.IsDead,
			}
		} else {
			// Чужие статы (скрываем лишнее)
			// Можно показывать HP bar, но не точные цифры, или только IsDead
			view.Stats = &api.StatsView{
				HP: target.Stats.HP, MaxHP: target.Stats.MaxHP, // HP бар нужен
				IsDead: target.Stats.IsDead,
				// Остальное (Stamina, Gold) будет 0/nil в JSON
			}
		}
	}

	// Если труп, меняем визуал здесь (или на клиенте)
	if isDead {
		view.Stats.IsDead = true
	}

	return view
}

func (s *GameService) executeCommand(cmd domain.InternalCommand, actor *domain.Entity) {
	handler, ok := s.handlers[cmd.Action]
	if !ok {
		return
	}

	allEntities := make([]*domain.Entity, 0, len(s.Entities)+1)
	allEntities = append(allEntities, s.Player)
	for i := range s.Entities {
		allEntities = append(allEntities, &s.Entities[i])
	}

	ctx := handlers.Context{
		World:    s.World,
		Entities: allEntities,
		Actor:    actor,
	}

	result, _ := handler(ctx, cmd.Payload)
	if result.Msg != "" {
		msgType := result.MsgType
		if msgType == "" {
			msgType = "INFO"
		}
		s.AddLog(result.Msg, msgType)
	}
}

// ... Остальные хелперы (getNextActor) ...
func (s *GameService) getNextActor() *domain.Entity {
	var activeEntities []*domain.Entity

	// Игрок
	if s.Player.AI != nil && !s.Player.Stats.IsDead {
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

func (s *GameService) AddLog(text, logType string) {
	s.Logs = append(s.Logs, api.LogEntry{
		ID: fmt.Sprintf("%d", time.Now().UnixNano()), Text: text, Type: logType, Timestamp: time.Now().UnixMilli(),
	})
}
