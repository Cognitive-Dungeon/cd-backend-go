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
	log.Println("[LOOP] Game Loop started")
	const MaxLoops = 1000
	loops := 0

	for {
		// 1. Кто ходит?
		activeActor := s.getNextActor()

		// Лог, чтобы видеть порядок ходов
		log.Printf("[LOOP] Active: %s (Tick: %d)", activeActor.Name, activeActor.NextActionTick)

		// 2. Обновляем время мира
		s.World.GlobalTick = activeActor.NextActionTick

		// 3. Ход Игрока
		if activeActor.ID == s.Player.ID {
			loops = 0 // Сбрасываем счетчик защиты, так как управление у человека

			log.Println("[LOOP] Waiting for Player command...")
			select {
			case cmd := <-s.CommandChan:
				log.Printf("[LOOP] Player Action: %s", cmd.Action)
				s.executeCommand(cmd)
			}
		} else {
			// 4. Ход NPC
			loops++
			if loops > MaxLoops {
				log.Println("[LOOP] Infinite loop detected! Forcing break.")
				time.Sleep(1 * time.Second) // Тормозим, чтобы не повесить CPU
				loops = 0
			}

			// Проверяем приоритетные команды (например, INIT)
			select {
			case cmd := <-s.CommandChan:
				if cmd.Action == "INIT" {
					s.executeCommand(cmd)
				}
			default:
				// Логика NPC
				s.processNPC(activeActor)

				// ОБЯЗАТЕЛЬНО: Отправляем обновление клиенту, чтобы он увидел ход врага
				s.publishUpdate()
			}
		}
	}
}

func (s *GameService) executeCommand(cmd domain.ClientCommand) {
	switch cmd.Action {
	case "INIT":
		s.AddLog("Добро пожаловать в Cognitive Dungeon.", "INFO")
		s.publishUpdate() // Сразу шлем ответ на INIT

	case "MOVE":
		var p domain.MovePayload
		if err := json.Unmarshal(cmd.Payload, &p); err == nil {
			s.handlePlayerMove(p.Dx, p.Dy)
		}

	case "WAIT":
		s.AddLog("Вы ждете...", "INFO")
		s.Player.NextActionTick += domain.TimeCostWait
		s.publishUpdate()

	case "TALK":
		// Заглушка для теста логов
		s.AddLog("Вы что-то бормочете.", "SPEECH")
		s.publishUpdate()
	}
}

// --- LOGIC ---

func (s *GameService) getNextActor() *domain.Entity {
	activeEntities := []*domain.Entity{s.Player}
	for i := range s.Entities {
		if !s.Entities[i].IsDead {
			activeEntities = append(activeEntities, &s.Entities[i])
		}
	}
	sort.Slice(activeEntities, func(i, j int) bool {
		return activeEntities[i].NextActionTick < activeEntities[j].NextActionTick
	})
	return activeEntities[0]
}

func (s *GameService) handlePlayerMove(dx, dy int) {
	res := systems.CalculateMove(s.Player, dx, dy, s.World, s.Entities)

	if res.BlockedBy != nil && res.BlockedBy.IsHostile {
		logMsg := systems.ApplyAttack(s.Player, res.BlockedBy)
		s.AddLog(logMsg, "COMBAT")
		s.Player.NextActionTick += domain.TimeCostAttackLight
	} else if res.HasMoved {
		s.Player.Pos.X = res.NewX
		s.Player.Pos.Y = res.NewY
		s.Player.NextActionTick += domain.TimeCostMove
	} else if res.IsWall {
		s.AddLog("Путь прегражден.", "ERROR")
	}

	// Важно: Мы НЕ вызываем publishUpdate здесь, так как он вызовется
	// в executeCommand или цикл прокрутится дальше.
}

func (s *GameService) processNPC(npc *domain.Entity) {
	action, target, dx, dy := systems.ComputeNPCAction(npc, s.Player, s.World, s.Entities)

	log.Printf("[NPC] %s decided to: %s", npc.Name, action)

	if action == "ATTACK" && target != nil {
		logMsg := systems.ApplyAttack(npc, target)
		s.AddLog(logMsg, "COMBAT")
		npc.NextActionTick += domain.TimeCostAttackLight
	} else if action == "MOVE" {
		npc.Pos.X += dx
		npc.Pos.Y += dy
		npc.NextActionTick += domain.TimeCostMove
	} else {
		// WAIT
		npc.NextActionTick += domain.TimeCostWait
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
