package engine

import (
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions"
	"cognitive-server/internal/engine/handlers/admin"
	"cognitive-server/internal/eventbus"
	"cognitive-server/internal/infrastructure/storage"
	"cognitive-server/internal/network"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/dungeon"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/utils"
	"encoding/json"
	"fmt"
	"math/rand"
)

type GameService struct {
	Config Config

	// Храним данные всех уровней (статические данные GameWorld)
	Worlds map[int]*domain.GameWorld

	// Активные запущенные инстансы
	Instances map[int]*Instance

	// Индекс: где находится сущность? (EntityID -> LevelID)
	EntityLocations map[domain.EntityID]int

	Storage *storage.ReplayService

	// Каналы для main.go (входная точка)
	JoinChan       chan *domain.Entity
	DisconnectChan chan domain.EntityID

	EventBus *eventbus.EventBus

	Hub *network.Broadcaster

	// Реестр хендлеров (общий для всех инстансов)
	actionHandlers map[domain.ActionType]handlers.HandlerFunc
	eventHandlers  map[domain.EventType]handlers.HandlerFunc
}

func NewService(cfg Config) *GameService {
	bus := eventbus.New(int(enums.EventTypeCount), eventbus.PanicRecovery())
	worlds, allEntities, seeds := buildInitialWorld(cfg.Seed)

	s := &GameService{
		Config:          cfg,
		Worlds:          worlds,
		Instances:       make(map[int]*Instance),
		EntityLocations: make(map[domain.EntityID]int),

		Storage: storage.NewReplayService("./replays"),

		JoinChan:       make(chan *domain.Entity, 10),
		DisconnectChan: make(chan domain.EntityID, 10),
		EventBus:       bus,

		Hub:            network.NewBroadcaster(),
		actionHandlers: make(map[domain.ActionType]handlers.HandlerFunc),
		eventHandlers:  make(map[domain.EventType]handlers.HandlerFunc),
	}

	s.registerHandlers()
	s.registerSystems()

	// 1. Создаем и запускаем Инстансы для каждого мира
	for id, world := range worlds {
		// Используем прекалькулированный сид
		instance := NewInstance(id, world, s, seeds[id])
		s.Instances[id] = instance
	}

	// 2. Распределяем начальные сущности по инстансам
	for _, e := range allEntities {
		if instance, ok := s.Instances[e.Level]; ok {
			s.EntityLocations[e.ID] = e.Level
			// Напрямую добавляем, так как циклы только запустились
			instance.addEntity(e)
		}
	}

	for _, instance := range s.Instances {
		go instance.Run()
	}

	return s
}

func (s *GameService) registerSystems() {
	// 1. Создаем системы
	moveSys := &systems.MovementSystem{}
	combatSys := &systems.CombatSystem{}

	// 2. Инициализируем их (подписка на события)
	moveSys.Init(s.EventBus)
	combatSys.Init(s.EventBus)

	// 3. Регистрируем слушатель для ЛОГОВ
	// Системы кидает EventTypeLogMessage, нам нужно поймать его и положить в нужный Instance
	eventbus.Subscribe(s.EventBus, eventbus.EventType(enums.EventTypeLogMessage), s.handleLogMessage)

	logger.Log.Info("Systems registered: Movement, Combat, Logging")
}

// GetEntity ищет сущность. Использует быстрый индекс EntityLocations.
func (s *GameService) GetEntity(id domain.EntityID) *domain.Entity {
	// 1. Узнаем уровень
	levelID, ok := s.EntityLocations[id]
	if !ok {
		return nil
	}

	// 2. Берем из мира (GameWorld хранит реестр)
	if world, ok := s.Worlds[levelID]; ok {
		return world.GetEntity(id)
	}
	return nil
}

func (s *GameService) registerHandlers() {
	s.actionHandlers[domain.ActionMove] = handlers.WithPayload(actions.HandleMove)
	s.actionHandlers[domain.ActionAttack] = handlers.WithPayload(actions.HandleAttack)
	s.actionHandlers[domain.ActionTalk] = handlers.WithPayload(actions.HandleTalk)
	s.actionHandlers[domain.ActionInteract] = handlers.WithPayload(actions.HandleInteract)
	s.actionHandlers[domain.ActionInit] = handlers.WithEmptyPayload(actions.HandleInit)
	s.actionHandlers[domain.ActionWait] = handlers.WithEmptyPayload(actions.HandleWait)

	// Inventory
	s.actionHandlers[domain.ActionPickup] = handlers.WithPayload(actions.HandlePickup)
	s.actionHandlers[domain.ActionDrop] = handlers.WithPayload(actions.HandleDrop)
	s.actionHandlers[domain.ActionUse] = handlers.WithPayload(actions.HandleUse)
	s.actionHandlers[domain.ActionEquip] = handlers.WithPayload(actions.HandleEquip)
	s.actionHandlers[domain.ActionUnequip] = handlers.WithPayload(actions.HandleUnequip)

	// Admin / Cheats
	s.actionHandlers[domain.ActionAdminTeleport] = handlers.WithPayload(admin.HandleTeleport)
	s.actionHandlers[domain.ActionAdminSpawn] = handlers.WithPayload(admin.HandleSpawn)
	s.actionHandlers[domain.ActionAdminHeal] = handlers.WithEmptyPayload(admin.HandleHeal)
	s.actionHandlers[domain.ActionAdminKill] = handlers.WithPayload(admin.HandleKill)
	s.actionHandlers[domain.ActionAdminOmni] = handlers.WithEmptyPayload(admin.HandleToggleOmni)
}

// Start теперь запускает только диспетчер входов/выходов
func (s *GameService) Start() {
	go s.DispatcherLoop()
}

// DispatcherLoop обрабатывает глобальные события входа/выхода
func (s *GameService) DispatcherLoop() {
	logger.Log.Info("Global Dispatcher started")

	for {
		select {
		// Новый игрок (из main.go)
		case newEntity := <-s.JoinChan:
			s.AddPlayerToLevel(newEntity)

		// Дисконнект (из main.go)
		case entityID := <-s.DisconnectChan:
			levelID, ok := s.EntityLocations[entityID]
			if ok {
				if instance, ok := s.Instances[levelID]; ok {
					// Сообщаем инстансу, чтобы он прервал ход
					select {
					case instance.LeaveChan <- entityID:
					default:
					}
				}
			}
		}
	}
}

// AddPlayerToLevel добавляет игрока в нужный инстанс
func (s *GameService) AddPlayerToLevel(e *domain.Entity) {
	instance, ok := s.Instances[e.Level]

	// Если уровня нет (например, процедурный левел, который еще не создан)
	// В текущей архитектуре мы создаем уровни при старте, но тут можно добавить Lazy Init
	if !ok {
		logger.Log.Warnf("Level %d not found for player %s", e.Level, e.ID)
		return
	}

	// Обновляем глобальный индекс
	s.EntityLocations[e.ID] = e.Level

	// Отправляем в инстанс
	instance.JoinChan <- e
}

// ProcessCommand маршрутизирует команды в нужный инстанс
func (s *GameService) ProcessCommand(cmd api.ClientCommand) {
	// 1. Где игрок?
	levelID, ok := s.EntityLocations[domain.EntityID(cmd.Token)]
	if !ok {
		// Игрока нет в индексе (возможно, только зашел и шлет INIT).
		// В этом случае игнорируем, так как INIT при входе отправляется автоматически из main.go,
		// но если клиент шлет повторный INIT вручную, он может потеряться.
		// Для надежности можно проверить JoinChan, но обычно это не нужно.
		return
	}

	// 2. Получаем инстанс
	instance, ok := s.Instances[levelID]
	if !ok {
		return
	}

	// 3. Формируем команду
	internalCmd := domain.InternalCommand{
		Action:  domain.ParseAction(cmd.Action),
		Token:   domain.EntityID(cmd.Token),
		Payload: cmd.Payload,
	}

	// 4. Находим объект актора (чтобы передать указатель, а не искать его снова внутри хода)
	// Используем быстрый поиск по миру
	actor := instance.World.GetEntity(domain.EntityID(cmd.Token))
	if actor == nil {
		return
	}

	// 5. Отправляем в канал инстанса
	instance.CommandChan <- InstanceCommand{
		Cmd:    internalCmd,
		Source: actor,
	}
}

// GetWorld гарантирует, что уровень существует и возвращает данные мира.
// Это безопасно для чтения из горутины логики (если мы только читаем компоненты).
func (s *GameService) GetWorld(levelID int) *domain.GameWorld {
	// 1. Если инстанс уже запущен — возвращаем мир
	if inst, ok := s.Instances[levelID]; ok {
		return inst.World
	}

	// 2. Если нет — генерируем на лету (как раньше было в ChangeLevel)
	// Важно: здесь мы создаем инстанс, но пока не запускаем его цикл, если это просто "подглядывание".
	// Но для простоты запустим сразу.

	logger.Log.Infof("Lazy generating level %d...", levelID)

	// Детерминированный сид
	levelSeed := s.Config.Seed + int64(levelID)

	// Генерация
	rng := rand.New(rand.NewSource(levelSeed))
	newWorld, newEntities, _ := dungeon.Generate(levelID, rng) // <-- Ваш генератор

	newInstance := NewInstance(levelID, newWorld, s, levelSeed)

	// Загрузка сущностей
	for i := range newEntities {
		newInstance.addEntity(&newEntities[i])
	}

	s.Instances[levelID] = newInstance
	go newInstance.Run() // Запускаем жизнь на уровне

	return newInstance.World
}

// Teleport выполняет "грязную работу" по переносу данных между инстансами.
func (s *GameService) Teleport(actor *domain.Entity, targetLevel int, targetPos domain.Position) {
	oldLevelID := actor.Level
	logger.Log.Infof("Teleporting %s: L%d -> L%d [%d,%d]", actor.ID, oldLevelID, targetLevel, targetPos.X, targetPos.Y)

	// 1. Удаляем из старого инстанса (Thread-safe via Channel)
	if oldInstance, ok := s.Instances[oldLevelID]; ok {
		oldInstance.LeaveChan <- actor.ID
	}

	// 2. Обновляем данные сущности
	// Важно: мы меняем данные "на лету", пока сущность "в лимбе" между каналами.
	actor.Level = targetLevel
	actor.Pos = targetPos

	// Сброс состояния AI и кэша зрения
	if actor.AI != nil {
		actor.AI.State = domain.AIStateIdle
		// В новом инстансе будет свое время (Tick), нужно синхронизироваться
		// Это сделает addEntity внутри инстанса
	}
	if actor.Vision != nil {
		actor.Vision.IsDirty = true
		actor.Vision.CachedVisibleTiles = nil
	}

	// 3. Обновляем глобальный индекс
	s.EntityLocations[actor.ID] = targetLevel

	// 4. Добавляем в новый инстанс
	// (GetWorld здесь уже не нужен, т.к. мы знаем, что он есть — Rule его вызывало)
	if newInstance, ok := s.Instances[targetLevel]; ok {
		newInstance.JoinChan <- actor
		newInstance.AddLog(fmt.Sprintf("%s прибывает на уровень.", actor.Name), "INFO")
	} else {
		// Edge case: Если Rule вызвал GetWorld, а инстанс исчез (маловероятно)
		// Восстанавливаем через GetWorld
		s.GetWorld(targetLevel)
		s.Instances[targetLevel].JoinChan <- actor
	}
}

// LoadReplay инициализирует сервис и один инстанс на основе файла реплея
func (s *GameService) LoadReplay(path string) error {
	// 1. Читаем файл
	session, err := s.Storage.Load(path)
	if err != nil {
		return err
	}

	logger.Log.Infof("Loaded replay: Seed=%d, Level=%d, Actions=%d", session.Seed, session.LevelID, len(session.Actions))

	// 2. Обновляем конфиг сервиса (чтобы MasterSeed совпадал)
	// В текущей реализации мы храним Seed в Instance, но глобальный тоже полезно обновить
	s.Config.Seed = session.Seed

	// 3. Воссоздаем мир с ТЕМ ЖЕ сидом
	// Важно: мы должны использовать логику генерации, аналогичную ChangeLevel/NewService
	// Для простоты предположим, что реплей записан для Dungeon (Level 1)
	// Если replay для Level 0 - logic similar.

	levelID := session.LevelID

	// Генерируем мир детерминировано
	rng := rand.New(rand.NewSource(session.Seed))

	var world *domain.GameWorld
	var entities []domain.Entity
	var startPos domain.Position

	if levelID == 0 {
		world, entities, startPos = dungeon.GenerateSurface()
	} else {
		world, entities, startPos = dungeon.Generate(levelID, rng)
	}

	// 4. Создаем Инстанс
	instance := NewInstance(levelID, world, s, session.Seed)
	// Важно: восстанавливаем actions и playerState в инстанс, чтобы если мы сохраним его снова, данные не потерялись
	instance.Replay.PlayerState = session.PlayerState

	// Загружаем сущности
	for i := range entities {
		instance.addEntity(&entities[i])
	}

	// Если в entities нет игрока (он приходит извне), его нужно создать.
	// В реплее actions[0] обычно содержит логин или первое действие игрока.
	// Для полной корректности нужно сохранять состояние игрока при входе в уровень.
	// ПОКА: Предполагаем, что игрок создается через CreatePlayer ("hero_1") и ставим его на старт.
	// Это упрощение. В продакшене реплей должен содержать snapshot игрока на входе.
	// TODO: Определиться откуда брать игрока и как это делать лучше всего

	// --- ВОССТАНОВЛЕНИЕ ИГРОКА ---
	var player *domain.Entity

	if len(session.PlayerState) > 0 {
		// ВАРИАНТ А: Снапшот есть (v2)
		logger.Log.Info("Restoring player from snapshot...")
		player = &domain.Entity{}
		if err := json.Unmarshal(session.PlayerState, player); err != nil {
			return fmt.Errorf("failed to restore player: %w", err)
		}

		// Принудительно ставим позицию, если она была записана криво, или доверяем снапшоту?
		// В снапшоте позиция с ПРЕДЫДУЩЕГО уровня. Нам нужно её обновить на стартовую для ЭТОГО уровня.
		player.Pos = startPos
		player.Level = levelID // Обновляем уровень

	} else {
		// ВАРИАНТ Б: Снапшота нет (v1 или уровень 0)
		logger.Log.Info("No snapshot found, creating fresh player...")
		playerID := "hero_1"
		playerSeed := utils.StringToSeed(playerID)
		playerRng := rand.New(rand.NewSource(playerSeed))
		player = dungeon.CreatePlayer(domain.EntityID(playerID), playerRng)
		player.Pos = startPos
	}
	// Задаем фейковый ControllerID, чтобы движок знал: этим персонажем управляет "внешняя сила" (реплей), а не AI.
	player.ControllerID = "replay_viewer"
	// ... логика поиска старта ...

	player.Pos = startPos
	player.Level = levelID
	instance.addEntity(player)
	s.EntityLocations[player.ID] = levelID

	// 5. Настраиваем режим воспроизведения
	instance.IsPlayback = true
	instance.PlaybackActions = session.Actions

	// Регистрируем инстанс
	s.Instances[levelID] = instance

	return nil
}

// StartPlayback запускает симуляцию загруженного инстанса
func (s *GameService) StartPlayback(levelID int) {
	if instance, ok := s.Instances[levelID]; ok {
		instance.RunSimulation()
	} else {
		logger.Log.Error("Instance not found for playback")
	}
}

func (s *GameService) handleLogMessage(ev domain.LogMessage) {
	// Нам нужно понять, к какому инстансу относится этот мир
	// Так как World внутри Instance уникален, ищем по LevelID
	if ev.World == nil {
		return
	}

	instance, ok := s.Instances[ev.World.Level]
	if !ok {
		// Если инстанс не активен (редкий кейс), просто пишем в консоль
		logger.Log.Warnf("[Orphan Log] %s: %s", ev.Type, ev.Text)
		return
	}

	instance.AddLog(ev.Text, ev.Type)
}
