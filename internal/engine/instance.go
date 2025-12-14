package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

// InstanceCommand обертка, чтобы передать команду и того, кто её вызвал
type InstanceCommand struct {
	Cmd    domain.InternalCommand
	Source *domain.Entity
}

// Instance представляет собой один изолированный запущенный уровень (игровую зону).
type Instance struct {
	ID    int               // ID уровня
	World *domain.GameWorld // Карта и SpatialHash

	// Локальные данные симуляции
	Entities    []*domain.Entity
	TurnManager *TurnManager

	// Каналы коммуникации
	CommandChan chan InstanceCommand // Команды от игроков
	JoinChan    chan *domain.Entity  // Вход новых игроков
	LeaveChan   chan string          // Выход/Смерть игроков

	// Ссылка на Service для доступа к Hub и глобальным настройкам
	Service *GameService

	CurrentTick int // Локальное время этого уровня

	Logs []api.LogEntry // Локальные логи уровня

	Rng    *rand.Rand            // Локальный генератор
	Seed   int64                 // Сид, с которого начался уровень
	Replay *domain.ReplaySession // Лента событий

}

func NewInstance(id int, world *domain.GameWorld, service *GameService, seed int64) *Instance {
	rngSource := rand.NewSource(seed)
	rng := rand.New(rngSource)
	return &Instance{
		ID:          id,
		World:       world,
		Entities:    make([]*domain.Entity, 0),
		TurnManager: NewTurnManager(),
		CommandChan: make(chan InstanceCommand, 100),
		JoinChan:    make(chan *domain.Entity, 10),
		LeaveChan:   make(chan string, 10),
		Service:     service,
		CurrentTick: 0,
		Logs:        []api.LogEntry{},
		Seed:        seed,
		Rng:         rng,
		Replay: &domain.ReplaySession{
			LevelID:   id,
			Seed:      seed,
			Timestamp: time.Now().Unix(),
			Actions:   make([]domain.ReplayAction, 0),
		},
	}
}

// Run запускает игровой цикл ЭТОГО инстанса.
func (i *Instance) Run() {
	logger.Log.WithField("instance_id", i.ID).Info("Instance loop started")

	for {
		// 1. Обработка входа/выхода (неблокирующая)
		select {
		case newEntity := <-i.JoinChan:
			i.addEntity(newEntity)
		case leftID := <-i.LeaveChan:
			i.removeEntity(leftID)
		default:
		}

		// 2. Кто ходит?
		item := i.TurnManager.PeekNext()
		if item == nil {
			time.Sleep(100 * time.Millisecond) // Спим, если уровень пуст
			continue
		}

		activeActor := item.Value
		i.CurrentTick = activeActor.AI.NextActionTick

		// 3. Проверка смерти
		if activeActor.Stats != nil && activeActor.Stats.IsDead {
			i.TurnManager.RemoveEntity(activeActor.ID)
			// Если был подписчик - обновляем ему экран
			if i.Service.Hub.HasSubscriber(activeActor.ID) {
				// Внимание: publishUpdate пока в Service, мы это поправим на след. шаге
				i.Service.publishUpdate(activeActor.ID, i)
			}
			continue
		}

		// 4. Рассылка состояния (тем, кто смотрит на этого актора)
		if i.Service.Hub.HasSubscriber(activeActor.ID) {
			i.Service.publishUpdate(activeActor.ID, i)
		}

		// 5. Логика хода
		isHuman := i.Service.Hub.HasSubscriber(activeActor.ID)

		if !isHuman {
			i.processAITurn(activeActor)
		} else {
			// --- ХОД ИГРОКА ---
			timeout := time.After(60 * time.Second)
			processed := false

			for !processed {
				select {
				// Вход во время ожидания
				case newEntity := <-i.JoinChan:
					i.addEntity(newEntity)

				// Выход во время ожидания
				case leftID := <-i.LeaveChan:
					i.removeEntity(leftID)
					if leftID == activeActor.ID {
						activeActor.AI.Wait(domain.TimeCostWait)
						processed = true
					}

				// Команда
				case wrapper := <-i.CommandChan:
					// Обрабатываем команду, если это активный игрок или системная команда
					if wrapper.Cmd.Token == activeActor.ID || wrapper.Cmd.Action == domain.ActionInit {
						i.executeCommand(wrapper.Cmd, wrapper.Source)
						if wrapper.Cmd.Action != domain.ActionInit {
							processed = true
						}
					}

				case <-timeout:
					logger.Log.WithFields(logrus.Fields{
						"instance": i.ID,
						"actor":    activeActor.ID,
					}).Warn("Turn timed out")
					activeActor.AI.Wait(domain.TimeCostWait)
					processed = true
				}
			}
		}

		// Обновляем приоритет в очереди
		i.TurnManager.UpdatePriority(activeActor.ID, activeActor.AI.NextActionTick)
	}
}

// addEntity добавляет сущность в структуры уровня
func (i *Instance) addEntity(e *domain.Entity) {
	i.Entities = append(i.Entities, e)
	i.World.RegisterEntity(e)
	i.World.AddEntity(e)

	if e.Stats != nil && !e.Stats.IsDead {
		// Синхронизация времени: берем время текущего активного или 0
		nextItem := i.TurnManager.PeekNext()
		if nextItem != nil {
			e.AI.NextActionTick = nextItem.Priority
		}
		i.TurnManager.AddEntity(e)
	}
}

// removeEntity удаляет сущность из уровня
func (i *Instance) removeEntity(id string) {
	// Удаляем из TurnManager
	i.TurnManager.RemoveEntity(id)

	// Удаляем из списка Entities
	for idx, e := range i.Entities {
		if e.ID == id {
			i.World.RemoveEntity(e) // Удаляет из карты и реестра

			// Удаляем из слайса (Swap with last)
			lastIdx := len(i.Entities) - 1
			i.Entities[idx] = i.Entities[lastIdx]
			i.Entities[lastIdx] = nil
			i.Entities = i.Entities[:lastIdx]
			break
		}
	}
}

// executeCommand выполняет команду в контексте уровня
func (i *Instance) executeCommand(cmd domain.InternalCommand, actor *domain.Entity) {
	// Проверяем, управляется ли актор агентом
	if actor.ControllerID != "" {
		i.recordAction(cmd, i.CurrentTick)
	}

	handler, ok := i.Service.actionHandlers[cmd.Action]
	if !ok {
		return
	}

	ctx := handlers.Context{
		Finder:   i.World, // Ищем только в этом мире!
		World:    i.World,
		Entities: i.Entities,
		Actor:    actor,
		Worlds:   i.Service.Worlds, // Для переходов (пока ссылаемся на глобальную мапу)

		// Для спавна новых сущностей (стрелы, суммоны)
		AddGlobalEntity: func(e *domain.Entity) {
			i.addEntity(e)
		},
		Switcher: i.Service,
		Rng:      i.Rng,
	}

	result, _ := handler(ctx, cmd.Payload)

	if result.Msg != "" {
		// Используем текущий AddLog (2 аргумента)
		i.AddLog(result.Msg, result.MsgType)
	}

	// События (переходы) пока оставляем на совести сервиса
	if result.Event != nil {
		i.Service.processEvent(actor, result.Event)
	}
}

func (i *Instance) recordAction(cmd domain.InternalCommand, tick int) {
	i.Replay.Actions = append(i.Replay.Actions, domain.ReplayAction{
		Tick:    tick,
		Token:   cmd.Token,
		Action:  cmd.Action,
		Payload: cmd.Payload,
	})
}

// processAITurn копия логики ИИ, адаптированная под Instance
func (i *Instance) processAITurn(npc *domain.Entity) {
	if npc.Stats != nil && npc.Stats.IsDead {
		return
	}
	if !npc.AI.IsHostile {
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	var target *domain.Entity
	minDist := 999.0

	// Ищем цель среди локальных сущностей i.Entities
	for _, other := range i.Entities {
		if other.ID == npc.ID || (other.Stats != nil && other.Stats.IsDead) {
			continue
		}

		isInanimate := other.Type == domain.EntityTypeItem || other.Type == domain.EntityTypeExit
		isSameType := other.Type == npc.Type

		if !isInanimate && !isSameType {
			dist := npc.Pos.DistanceTo(other.Pos)
			if dist < minDist {
				minDist = dist
				target = other
			}
		}
	}

	if target == nil {
		npc.AI.Wait(domain.TimeCostWait)
		return
	}

	action, _, dx, dy := systems.ComputeNPCAction(npc, target, i.World, i.Rng)

	switch action {
	case domain.ActionAttack:
		payload, _ := json.Marshal(api.EntityPayload{TargetID: target.ID})
		i.executeCommand(domain.InternalCommand{Action: domain.ActionAttack, Token: npc.ID, Payload: payload}, npc)
	case domain.ActionMove:
		payload, _ := json.Marshal(api.DirectionPayload{Dx: dx, Dy: dy})
		i.executeCommand(domain.InternalCommand{Action: domain.ActionMove, Token: npc.ID, Payload: payload}, npc)
	default:
		npc.AI.Wait(domain.TimeCostWait)
	}
}
