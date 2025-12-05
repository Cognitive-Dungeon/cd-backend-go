package core

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/dungeon"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"
)

type GameService struct {
	World    *domain.GameWorld
	Player   *domain.Entity
	Entities []domain.Entity
	Logs     []domain.LogEntry

	CommandChan chan domain.ClientCommand
	Hub         *Broadcaster
}

func NewService() *GameService {
	world, entities, startPos := dungeon.Generate(1)
	player := &domain.Entity{
		ID:     "p1",
		Name:   "Герой",
		Symbol: "@",
		Color:  "text-cyan-400",
		Type:   domain.EntityTypePlayer,
		Pos:    startPos,
		Stats: domain.Stats{
			HP: 100, MaxHP: 100, Stamina: 100, MaxStamina: 100, Gold: 50, Strength: 10,
		},
	}

	return &GameService{
		World:       world,
		Player:      player,
		Entities:    entities,
		Logs:        []domain.LogEntry{},
		CommandChan: make(chan domain.ClientCommand, 100),
		Hub:         NewBroadcaster(),
	}
}

func (s *GameService) Start() {
	go s.RunGameLoop()
}

func (s *GameService) ProcessCommand(cmd domain.ClientCommand) {
	select {
	case s.CommandChan <- cmd:
	default:
		log.Println("[WARN] Command queue full")
	}
}

func (s *GameService) publishUpdate() {
	// Копируем логи
	currentLogs := make([]domain.LogEntry, len(s.Logs))
	copy(currentLogs, s.Logs)
	s.Logs = []domain.LogEntry{}

	response := domain.ServerResponse{
		Type:     "UPDATE",
		World:    s.World,
		Player:   s.Player,
		Entities: s.Entities,
		Logs:     currentLogs,
	}

	log.Printf("[BROADCAST] Sending update. Logs count: %d", len(currentLogs))
	s.Hub.Broadcast(response)
}

func (s *GameService) GetState() *domain.ServerResponse {
	currentLogs := make([]domain.LogEntry, len(s.Logs))
	copy(currentLogs, s.Logs)
	s.Logs = []domain.LogEntry{}

	return &domain.ServerResponse{
		Type:     "UPDATE",
		World:    s.World,
		Player:   s.Player,
		Entities: s.Entities,
		Logs:     currentLogs,
	}
}

// --- ГЛАВНЫЙ ЦИКЛ ---

func (s *GameService) RunGameLoop() {
	log.Println("[LOOP] Arbiter Loop started")

	for {
		// 1. Кто ходит?
		activeActor := s.getNextActor()
		s.World.GlobalTick = activeActor.NextActionTick

		// 2. Уведомляем всех: "Сейчас ход ID=..."
		s.publishTurn(activeActor.ID)

		// 3. Ждем команду ИМЕННО от этого актера
		timeout := time.After(5 * time.Second) // Таймаут на ход (чтобы игра не зависла)

		commandProcessed := false

		for !commandProcessed {
			select {
			case cmd := <-s.CommandChan:
				// Хак для MVP: Если токен пустой, считаем что это Игрок (p1)
				// В будущем тут будет проверка сессий
				senderID := cmd.Token
				if senderID == "" {
					senderID = "p1"
				}

				// Если команда от того, чей сейчас ход (или системная INIT)
				if senderID == activeActor.ID || cmd.Action == "INIT" {
					if cmd.Action == "INIT" {
						s.executeCommand(cmd, s.Player) // INIT всегда от игрока пока
					} else {
						s.executeCommand(cmd, activeActor)
						commandProcessed = true // Выходим из ожидания, переходим к следующему
					}
				} else {
					log.Printf("[ARBITER] Ignored command from %s (it is %s's turn)", senderID, activeActor.Name)
				}

			case <-timeout:
				log.Printf("[ARBITER] Timeout for %s. Forcing WAIT.", activeActor.Name)
				activeActor.NextActionTick += domain.TimeCostWait
				commandProcessed = true
			}
		}

		// Рассылаем результат хода
		s.publishUpdate()
	}
}

// Метод для рассылки уведомления о ходе
func (s *GameService) publishTurn(activeID string) {
	// Можно оптимизировать и слать только ID, но пока шлем апдейт с флагом
	s.publishUpdateWithActive(activeID)
}

func (s *GameService) publishUpdateWithActive(activeID string) {
	// Копирование логов...
	currentLogs := make([]domain.LogEntry, len(s.Logs))
	copy(currentLogs, s.Logs)
	s.Logs = []domain.LogEntry{}

	response := domain.ServerResponse{
		Type:           "UPDATE",
		World:          s.World,
		Player:         s.Player,
		Entities:       s.Entities,
		Logs:           currentLogs,
		ActiveEntityID: activeID, // <--- Важное поле
	}
	s.Hub.Broadcast(response)
}

func (s *GameService) executeCommand(cmd domain.ClientCommand, actor *domain.Entity) {
	switch cmd.Action {
	case "INIT":
		s.AddLog("Добро пожаловать в Cognitive Dungeon.", "INFO")

	case "MOVE":
		var p domain.DirectionPayload
		if err := json.Unmarshal(cmd.Payload, &p); err == nil {
			s.handleMove(actor, p.Dx, p.Dy)
		}

	case "ATTACK":
		var p domain.EntityPayload
		if err := json.Unmarshal(cmd.Payload, &p); err == nil {
			s.handleAttack(actor, p.TargetID)
		}

	case "WAIT":
		s.AddLog(fmt.Sprintf("%s пропускает ход.", actor.Name), "INFO")
		actor.NextActionTick += domain.TimeCostWait

	case "TALK":
		var p domain.EntityPayload
		// Если payload пустой (крик в пустоту)
		if err := json.Unmarshal(cmd.Payload, &p); err == nil && p.TargetID != "" {
			// TODO: s.handleTalk(actor, p.TargetID) В будущем
			s.AddLog(fmt.Sprintf("Вы говорите с %s", p.TargetID), "SPEECH")
		} else {
			s.AddLog("Вы бормочете в пустоту.", "SPEECH")
		}
	}
}

// --- LOGIC ---

func (s *GameService) getNextActor() *domain.Entity {
	// 1. Собираем всех ЖИВЫХ и АКТИВНЫХ
	// Игрок всегда активен
	activeEntities := []*domain.Entity{s.Player}

	for i := range s.Entities {
		e := &s.Entities[i]
		// Фильтр: Мертвые и Пассивные объекты не участвуют в очереди
		if !e.IsDead && (e.Type == domain.EntityTypeNPC || e.Type == domain.EntityTypeEnemy) {
			activeEntities = append(activeEntities, e)
		}
	}

	// 2. Сортируем: кто меньше ждал, тот и ходит
	sort.Slice(activeEntities, func(i, j int) bool {
		return activeEntities[i].NextActionTick < activeEntities[j].NextActionTick
	})

	return activeEntities[0]
}

func (s *GameService) handleMove(actor *domain.Entity, dx, dy int) {
	res := systems.CalculateMove(actor, dx, dy, s.World, s.Entities)

	if res.BlockedBy != nil && res.BlockedBy.IsHostile != actor.IsHostile {
		// Атака при столкновении
		logMsg := systems.ApplyAttack(actor, res.BlockedBy)
		s.AddLog(logMsg, "COMBAT")
		actor.NextActionTick += domain.TimeCostAttackLight
	} else if res.HasMoved {
		actor.Pos.X = res.NewX
		actor.Pos.Y = res.NewY
		actor.NextActionTick += domain.TimeCostMove
	} else {
		// Если уперся в стену.
		// Пишем ошибку только для игрока, чтобы не спамить логами тупых ботов
		if actor.Type == domain.EntityTypePlayer {
			s.AddLog("Путь прегражден.", "ERROR")
		} else {
			// Бота штрафуем, чтобы он не ддосил сервер попытками пройти сквозь стену
			actor.NextActionTick += domain.TimeCostWait
		}
	}
}

func (s *GameService) handleAttack(actor *domain.Entity, targetID string) {
	// Найти цель по ID
	var target *domain.Entity
	for i := range s.Entities {
		if s.Entities[i].ID == targetID {
			target = &s.Entities[i]
			break
		}
	}
	if s.Player.ID == targetID {
		target = s.Player
	}

	if target != nil {
		logMsg := systems.ApplyAttack(actor, target)
		s.AddLog(logMsg, "COMBAT")
		actor.NextActionTick += domain.TimeCostAttackLight
	}
}

func (s *GameService) AddLog(text, logType string) {
	s.Logs = append(s.Logs, domain.LogEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:      text,
		Type:      logType,
		Timestamp: time.Now().UnixMilli(),
	})
}
