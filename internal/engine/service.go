package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions"
	"cognitive-server/internal/engine/handlers/admin"
	"cognitive-server/internal/engine/handlers/events"
	"cognitive-server/internal/infrastructure/storage"
	"cognitive-server/internal/network"
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

	Hub *network.Broadcaster

	// Реестр хендлеров (общий для всех инстансов)
	actionHandlers map[domain.ActionType]handlers.HandlerFunc
	eventHandlers  map[domain.EventType]handlers.HandlerFunc
}

func NewService(cfg Config) *GameService {
	worlds, allEntities, seeds := buildInitialWorld(cfg.Seed)

	s := &GameService{
		Config:          cfg,
		Worlds:          worlds,
		Instances:       make(map[int]*Instance),
		EntityLocations: make(map[domain.EntityID]int),

		Storage: storage.NewReplayService("./replays"),

		JoinChan:       make(chan *domain.Entity, 10),
		DisconnectChan: make(chan domain.EntityID, 10),

		Hub:            network.NewBroadcaster(),
		actionHandlers: make(map[domain.ActionType]handlers.HandlerFunc),
		eventHandlers:  make(map[domain.EventType]handlers.HandlerFunc),
	}

	s.registerHandlers()

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

	s.eventHandlers[domain.EventLevelTransition] = handlers.WithPayload(events.HandleLevelTransition)

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

func (s *GameService) ChangeLevel(actor *domain.Entity, newLevelID int, targetPosID domain.EntityID) {
	oldLevelID := actor.Level

	logger.Log.Infof("Transitioning entity %s from Level %d to %d", actor.ID, oldLevelID, newLevelID)

	// Сохраняем состояние игрока ПЕРЕД тем, как он попадет в новый мир.
	// Это состояние будет записано в заголовок реплея нового уровня.
	var playerSnapshot json.RawMessage
	if serialized, err := json.Marshal(actor); err == nil {
		playerSnapshot = serialized
	} else {
		logger.Log.Errorf("Failed to snapshot player: %v", err)
	}

	// 1. Получаем (или создаем) целевой Инстанс
	newInstance, ok := s.Instances[newLevelID]
	if !ok {
		logger.Log.Infof("Generating new level %d on the fly...", newLevelID)

		levelSeed := s.Config.Seed + int64(newLevelID)

		rng := rand.New(rand.NewSource(levelSeed))
		newWorld, newEntities, _ := dungeon.Generate(newLevelID, rng)

		newInstance = NewInstance(newLevelID, newWorld, s, levelSeed)
		newInstance.Replay.PlayerState = playerSnapshot

		for i := range newEntities {
			newInstance.addEntity(&newEntities[i])
		}

		s.Instances[newLevelID] = newInstance
		go newInstance.Run()
	}

	// 2. Удаляем актора из СТАРОГО инстанса
	if oldInstance, ok := s.Instances[oldLevelID]; ok {
		oldInstance.LeaveChan <- actor.ID
	}

	// 3. Вычисляем позицию в НОВОМ инстансе
	targetPos := domain.Position{X: 1, Y: 1}

	// Ищем в реестре мира (это безопасно, т.к. GameWorld - это данные)
	targetEntity := newInstance.World.GetEntity(targetPosID)
	if targetEntity != nil {
		targetPos = targetEntity.Pos
	} else {
		// Fallback: центр карты
		cx, cy := newInstance.World.Width/2, newInstance.World.Height/2
		if !newInstance.World.Map[cy][cx].IsWall {
			targetPos = domain.Position{X: cx, Y: cy}
		}
	}

	// 4. Обновляем данные актора
	actor.Level = newLevelID
	actor.Pos = targetPos

	if actor.AI != nil {
		actor.AI.State = domain.AIStateIdle
		// Синхронизация времени
		actor.AI.NextActionTick = newInstance.CurrentTick
	}

	// Invalidate FOV
	if actor.Vision != nil {
		actor.Vision.IsDirty = true
		actor.Vision.CachedVisibleTiles = nil // Force clear old map
	}

	// 5. Обновляем Глобальный Индекс
	s.EntityLocations[actor.ID] = newLevelID

	// 6. Добавляем актора в НОВЫЙ инстанс
	newInstance.JoinChan <- actor

	newInstance.AddLog(fmt.Sprintf("%s переходит на уровень %d.", actor.Name, newLevelID), "INFO")
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
