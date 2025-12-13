package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/engine/handlers/actions"
	"cognitive-server/internal/engine/handlers/admin"
	"cognitive-server/internal/engine/handlers/events"
	"cognitive-server/internal/network"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/dungeon"
	"cognitive-server/pkg/logger"
	"fmt"
	"math/rand"
	"time"
)

type GameService struct {
	// Храним данные всех уровней (статические данные GameWorld)
	Worlds map[int]*domain.GameWorld

	// Активные запущенные инстансы
	Instances map[int]*Instance

	// Индекс: где находится сущность? (EntityID -> LevelID)
	EntityLocations map[string]int

	// Каналы для main.go (входная точка)
	JoinChan       chan *domain.Entity
	DisconnectChan chan string

	Hub *network.Broadcaster

	// Реестр хендлеров (общий для всех инстансов)
	actionHandlers map[domain.ActionType]handlers.HandlerFunc
	eventHandlers  map[domain.EventType]handlers.HandlerFunc
}

func NewService() *GameService {
	worlds, allEntities := buildInitialWorld()

	s := &GameService{
		Worlds:          worlds,
		Instances:       make(map[int]*Instance),
		EntityLocations: make(map[string]int),

		JoinChan:       make(chan *domain.Entity, 10),
		DisconnectChan: make(chan string, 10),

		Hub:            network.NewBroadcaster(),
		actionHandlers: make(map[domain.ActionType]handlers.HandlerFunc),
		eventHandlers:  make(map[domain.EventType]handlers.HandlerFunc),
	}

	s.registerHandlers()

	// 1. Создаем и запускаем Инстансы для каждого мира
	for id, world := range worlds {
		instance := NewInstance(id, world, s)
		s.Instances[id] = instance

		// Запускаем игровой цикл этого уровня в отдельной горутине
		go instance.Run()
	}

	// 2. Распределяем начальные сущности по инстансам
	for _, e := range allEntities {
		if instance, ok := s.Instances[e.Level]; ok {
			s.EntityLocations[e.ID] = e.Level
			// Напрямую добавляем, так как циклы только запустились
			instance.addEntity(e)
		}
	}

	return s
}

// GetEntity ищет сущность. Использует быстрый индекс EntityLocations.
func (s *GameService) GetEntity(id string) *domain.Entity {
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
	levelID, ok := s.EntityLocations[cmd.Token]
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
		Token:   cmd.Token,
		Payload: cmd.Payload,
	}

	// 4. Находим объект актора (чтобы передать указатель, а не искать его снова внутри хода)
	// Используем быстрый поиск по миру
	actor := instance.World.GetEntity(cmd.Token)
	if actor == nil {
		return
	}

	// 5. Отправляем в канал инстанса
	instance.CommandChan <- InstanceCommand{
		Cmd:    internalCmd,
		Source: actor,
	}
}

func (s *GameService) ChangeLevel(actor *domain.Entity, newLevelID int, targetPosID string) {
	oldLevelID := actor.Level

	logger.Log.Infof("Transitioning entity %s from Level %d to %d", actor.ID, oldLevelID, newLevelID)

	// 1. Получаем (или создаем) целевой Инстанс
	newInstance, ok := s.Instances[newLevelID]
	if !ok {
		logger.Log.Infof("Generating new level %d on the fly...", newLevelID)

		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		newWorld, newEntities, _ := dungeon.Generate(newLevelID, rng)

		newInstance = NewInstance(newLevelID, newWorld, s)

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
		actor.AI.State = "IDLE"
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
