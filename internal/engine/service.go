package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions"
	"cognitive-server/internal/network"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/dungeon"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sort"
	"time"
)

type GameService struct {
	// Worlds хранит все загруженные/сгенерированные уровни
	Worlds map[int]*domain.GameWorld

	GlobalTick int

	// Entities хранит указатели на ВСЕ сущности (Игроки, NPC, Монстры)
	Entities       []*domain.Entity
	EntityRegistry map[string]*domain.Entity

	Logs []api.LogEntry

	CommandChan chan domain.InternalCommand
	Hub         *network.Broadcaster

	handlers map[domain.ActionType]handlers.HandlerFunc
}

func NewService() *GameService {
	worlds := make(map[int]*domain.GameWorld)
	var allEntities []*domain.Entity
	entityRegistry := make(map[string]*domain.Entity)

	// Создаем генератор случайных чисел
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Генерируем поверхность (уровень 0)
	surfaceWorld, surfaceEntities, startPos := dungeon.GenerateSurface()
	worlds[0] = surfaceWorld

	// Генерируем первый уровень подземелья (уровень 1)
	dungeonWorld1, dungeonEntities1, _ := dungeon.Generate(1, rng)
	worlds[1] = dungeonWorld1

	// --- Создание Героя ---
	player := &domain.Entity{
		ID:    "hero_1",
		Name:  "Герой",
		Type:  domain.EntityTypePlayer,
		Pos:   startPos,
		Level: 0,

		Render: &domain.RenderComponent{Symbol: "@", Color: "#22D3EE", Label: "A"},
		Stats: &domain.StatsComponent{
			HP: 100, MaxHP: 100, Stamina: 100, MaxStamina: 100, Gold: 50, Strength: 10,
		},
		AI:        &domain.AIComponent{NextActionTick: 0, IsHostile: false}, // AI компонент нужен для очереди ходов
		Narrative: &domain.NarrativeComponent{Description: "Искатель приключений."},
		Vision:    &domain.VisionComponent{Radius: domain.VisionRadius},
		Memory:    &domain.MemoryComponent{ExploredIDs: make(map[int]bool)},
	}
	allEntities = append(allEntities, player)

	// --- Добавление сгенерированных сущностей (враги, предметы) ---
	for i := range surfaceEntities {
		allEntities = append(allEntities, &surfaceEntities[i])
	}
	for i := range dungeonEntities1 {
		allEntities = append(allEntities, &dungeonEntities1[i])
	}

	s := &GameService{
		Worlds:         worlds,
		Entities:       allEntities,
		EntityRegistry: entityRegistry,
		GlobalTick:     0,
		Logs:           []api.LogEntry{},
		CommandChan:    make(chan domain.InternalCommand, 100),
		Hub:            network.NewBroadcaster(),
		handlers:       make(map[domain.ActionType]handlers.HandlerFunc),
	}

	// 4. Регистрация всех сущностей в мире
	for _, e := range allEntities {
		// Добавляем в глобальный реестр
		s.EntityRegistry[e.ID] = e

		// Добавляем в SpatialHash нужного мира
		if world, ok := s.Worlds[e.Level]; ok {
			world.AddEntity(e)
		}
	}

	s.registerHandlers()
	return s
}

// GetEntity ищет сущность по ID во всех загруженных мирах.
func (s *GameService) GetEntity(id string) *domain.Entity {
	return s.EntityRegistry[id]
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
		log.Printf("Unknown action: %s", externalCmd.Action)
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
	log.Println("[LOOP] Game Loop started")

	for {
		// Если в хабе нет ни одного подписчика (ни игроков, ни ботов),
		// ставим симуляцию на паузу, чтобы не тратить ресурсы и не "прокручивать" время.
		if s.Hub.SubscriberCount() == 0 {
			time.Sleep(100 * time.Millisecond) // Небольшая задержка, чтобы не загружать CPU
			continue
		}

		// 1. Кто ходит следующим?
		activeActor := s.getNextActor()

		// Если никого нет (пустой мир или все мертвы), ждем и повторяем
		if activeActor == nil {
			time.Sleep(1 * time.Second)
			continue
		}

		log.Printf("[LOOP] Next actor: %s (%s) | NextActionTick: %d", activeActor.Name, activeActor.ID, activeActor.AI.NextActionTick)

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
				log.Printf("[TIMEOUT] %s (%s) skips turn.", activeActor.Name, activeActor.ID)
				activeActor.AI.Wait(domain.TimeCostWait)
				commandProcessed = true
			}
		}
	}
}

// processAITurn обрабатывает логику NPC
func (s *GameService) processAITurn(npc *domain.Entity) {
	if !npc.AI.IsHostile {
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	var target *domain.Entity
	minDist := 999.0

	// 1. Ищем ближайшую цель на ТОМ ЖЕ УРОВНЕ
	for _, other := range s.Entities {
		if other.Level != npc.Level {
			continue // Игнорируем сущностей на других уровнях
		}
		if other.ID == npc.ID || (other.Stats != nil && other.Stats.IsDead) {
			continue // Игнорируем себя и мертвых
		}

		// Агрессия на Игроков
		if other.Type == domain.EntityTypePlayer {
			dist := npc.Pos.DistanceTo(other.Pos)
			if dist < minDist {
				minDist = dist
				target = other
			}
		}
	}

	// 2. Если целей на этом уровне нет, ждем
	if target == nil {
		npc.AI.Wait(domain.TimeCostWait)
		return // ВАЖНО: Выходим, если нет цели. Не вызываем AI.
	}

	// 3. Получаем мир, в котором находится NPC
	npcWorld, ok := s.Worlds[npc.Level]
	if !ok {
		log.Printf("[ERROR] NPC %s is on a non-existent level %d. Waiting.", npc.ID, npc.Level)
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	// 4. Вычисляем действие
	action, _, dx, dy := systems.ComputeNPCAction(npc, target, npcWorld)

	// 5. Конвертируем решение AI во внутреннюю команду
	switch action {
	case domain.ActionAttack:
		payload, _ := json.Marshal(api.EntityPayload{TargetID: target.ID})
		s.executeCommand(domain.InternalCommand{
			Action:  domain.ActionAttack,
			Token:   npc.ID,
			Payload: payload,
		}, npc)

	case domain.ActionMove:
		payload, _ := json.Marshal(api.DirectionPayload{Dx: dx, Dy: dy})
		s.executeCommand(domain.InternalCommand{
			Action:  domain.ActionMove,
			Token:   npc.ID,
			Payload: payload,
		}, npc)

	default:
		// Wait
		npc.AI.Wait(domain.TimeCostWait)
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
		log.Printf("[ERROR] Actor %s is on a non-existent level %d", actor.ID, actor.Level)
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

func (s *GameService) processEvent(actor *domain.Entity, eventData json.RawMessage) {
	var genericEvent struct {
		Event string `json:"event"`
	}
	if err := json.Unmarshal(eventData, &genericEvent); err != nil {
		log.Printf("Error parsing event: %v", err)
		return
	}

	switch genericEvent.Event {
	case "LEVEL_TRANSITION":
		s.handleLevelTransition(actor, eventData)
	// Здесь в будущем могут быть другие события: "SPAWN_MONSTER", "OPEN_DOOR", etc.
	default:
		log.Printf("Unknown event type: %s", genericEvent.Event)
	}
}

// --- НОВАЯ ФУНКЦИЯ: Логика перехода между уровнями ---
func (s *GameService) handleLevelTransition(actor *domain.Entity, eventData json.RawMessage) {
	var transitionEvent struct {
		TargetLevel int    `json:"targetLevel"`
		TargetPosId string `json:"targetPosId"`
	}
	if err := json.Unmarshal(eventData, &transitionEvent); err != nil {
		log.Printf("Error parsing LEVEL_TRANSITION event: %v", err)
		return
	}

	oldLevel := actor.Level
	newLevel := transitionEvent.TargetLevel

	// 1. Находим старый и новый миры
	oldWorld, okOld := s.Worlds[oldLevel]
	newWorld, okNew := s.Worlds[newLevel]

	if !okOld {
		log.Printf("Actor %s tried to transition from a non-existent level %d", actor.ID, oldLevel)
		return
	}

	// Если нового мира нет - генерируем его на лету
	if !okNew {
		log.Printf("Generating new level on the fly: %d", newLevel)
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		generatedWorld, newEntities, _ := dungeon.Generate(newLevel, rng)
		s.Worlds[newLevel] = generatedWorld
		newWorld = generatedWorld
		// Регистрируем новых сущностей
		for i := range newEntities {
			e := &newEntities[i]
			s.Entities = append(s.Entities, e)
			newWorld.AddEntity(e)
		}
	}

	// 2. Находим целевую позицию в новом мире
	var targetPos domain.Position
	targetEntity := s.GetEntity(transitionEvent.TargetPosId)
	if targetEntity != nil {
		targetPos = targetEntity.Pos
	} else {
		log.Printf("Could not find target position entity %s on level %d. Placing at default.", transitionEvent.TargetPosId, newLevel)
		targetPos = domain.Position{X: newWorld.Width / 2, Y: newWorld.Height / 2} // Запасной вариант
	}

	// 3. Перемещаем актора
	oldWorld.RemoveEntity(actor) // Удаляем из старого мира
	actor.Level = newLevel       // Меняем уровень
	actor.Pos = targetPos        // Меняем позицию
	newWorld.AddEntity(actor)    // Добавляем в новый мир

	// 4. Логирование
	if newLevel > oldLevel {
		s.AddLog(fmt.Sprintf("%s спускается глубже...", actor.Name), "INFO")
	} else {
		s.AddLog(fmt.Sprintf("%s поднимается наверх...", actor.Name), "INFO")
	}
}

// publishUpdate рассылает состояние ВСЕМ подключенным сущностям
func (s *GameService) publishUpdate(activeID string) {
	// Пробегаем по всем сущностям, и если у них есть "душа" (подключенный клиент), шлем апдейт
	for _, e := range s.Entities {
		if s.Hub.HasSubscriber(e.ID) {
			state := s.BuildStateFor(e, activeID)
			s.Hub.SendTo(e.ID, *state)
		}
	}

	// Очищаем логи ПОСЛЕ рассылки (так как они рассылаются всем одинаковые в текущей итерации)
	// Примечание: В production лучше хранить лог буфер или рассылать события сразу.
	s.Logs = []api.LogEntry{}
}

// BuildStateFor создает персональный слепок мира для observer
func (s *GameService) BuildStateFor(observer *domain.Entity, activeID string) *api.ServerResponse {
	observerWorld, ok := s.Worlds[observer.Level]
	if !ok {
		return &api.ServerResponse{Type: "ERROR", Logs: []api.LogEntry{{Text: "You are in the void."}}}
	}

	// 1. Расчет FOV (Поля зрения)
	var visibleIdxs map[int]bool
	isGod := false

	if observer.Vision != nil {
		visibleIdxs = systems.ComputeVisibleTiles(observerWorld, observer.Pos, observer.Vision)
		if visibleIdxs == nil { // nil возвращается для Omniscient (всевидящих)
			isGod = true
		}
	}

	// Обновляем память (туман войны)
	if observer.Memory != nil && !isGod && visibleIdxs != nil {
		for idx := range visibleIdxs {
			observer.Memory.ExploredIDs[idx] = true
		}
	}

	// 2. Формирование карты (Map DTO)
	var mapDTO []api.TileView
	// TODO: Оптимизация: можно отправлять только изменения, но пока шлем всю видимую карту
	for y := 0; y < observerWorld.Height; y++ {
		for x := 0; x < observerWorld.Width; x++ {
			idx := observerWorld.GetIndex(x, y)

			// Проверяем, знает ли наблюдатель об этой клетке
			isExplored := isGod
			if !isGod && observer.Memory != nil {
				isExplored = observer.Memory.ExploredIDs[idx]
			}

			// Если клетка исследована, добавляем её в ответ
			if isExplored {
				tile := observerWorld.Map[y][x]
				isVisible := isGod || visibleIdxs[idx]

				tView := api.TileView{
					X: x, Y: y, IsWall: tile.IsWall,
					IsVisible:  isVisible,
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

	// 3. Формирование списка сущностей (Entities DTO)
	var viewEntities []api.EntityView

	for _, e := range s.Entities {
		// --- ВАЖНО: Показываем только сущностей на том же уровне ---
		if e.Level != observer.Level {
			continue
		}

		// Себя видим всегда
		if e.ID == observer.ID {
			viewEntities = append(viewEntities, s.toEntityView(e, observer))
			continue
		}

		// Остальных - если они в зоне видимости
		idx := observerWorld.GetIndex(e.Pos.X, e.Pos.Y)
		if isGod || visibleIdxs[idx] {
			viewEntities = append(viewEntities, s.toEntityView(e, observer))
		}
	}

	// Копия логов, чтобы не было гонки данных
	logsCopy := make([]api.LogEntry, len(s.Logs))
	copy(logsCopy, s.Logs)

	return &api.ServerResponse{
		Type:           "UPDATE",
		Tick:           s.GlobalTick,
		MyEntityID:     observer.ID,
		ActiveEntityID: activeID,
		Grid:           &api.GridMeta{Width: observerWorld.Width, Height: observerWorld.Height},
		Map:            mapDTO,
		Entities:       viewEntities,
		Logs:           logsCopy,
	}
}

// toEntityView конвертирует доменную сущность в DTO с учетом прав доступа (observer)
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

	// Логика видимости статов
	isMe := target.ID == observer.ID
	isDead := target.Stats != nil && target.Stats.IsDead

	if target.Stats != nil {
		if isMe {
			// Владелец видит всё
			view.Stats = &api.StatsView{
				HP: target.Stats.HP, MaxHP: target.Stats.MaxHP,
				Stamina: target.Stats.Stamina, MaxStamina: target.Stats.MaxStamina,
				Gold: target.Stats.Gold, Strength: target.Stats.Strength,
				IsDead: target.Stats.IsDead,
			}
		} else {
			// Чужаки видят минимум (можно добавить Perception Check здесь)
			view.Stats = &api.StatsView{
				HP: target.Stats.HP, MaxHP: target.Stats.MaxHP,
				IsDead: target.Stats.IsDead,
			}
		}
	}

	if isDead {
		view.Stats.IsDead = true
	}

	return view
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
}
