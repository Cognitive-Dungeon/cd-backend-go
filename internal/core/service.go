package core

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/dungeon"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

type GameService struct {
	World    *domain.GameWorld
	Player   *domain.Entity
	Entities []domain.Entity
	Logs     []domain.LogEntry
}

func NewService() *GameService {
	// Генерация мира
	world, entities, startPos := dungeon.Generate(1)

	// Создание игрока
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
		World:    world,
		Player:   player,
		Entities: entities,
		Logs:     []domain.LogEntry{},
	}
}

// ProcessCommand - Обработка ввода от клиента
func (s *GameService) ProcessCommand(cmd domain.ClientCommand) *domain.ServerResponse {
	s.Logs = []domain.LogEntry{} // Очистка старых логов
	response := &domain.ServerResponse{Type: "UPDATE"}

	playerActed := false

	switch cmd.Action {
	case "INIT":
		s.AddLog("Добро пожаловать в Cognitive Dungeon.", "INFO")
		response.Type = "INIT"

	case "MOVE":
		var p domain.MovePayload
		if err := json.Unmarshal(cmd.Payload, &p); err == nil {
			if s.handlePlayerMove(p.Dx, p.Dy) {
				playerActed = true
			}
		}

	case "WAIT":
		s.AddLog("Вы ждете...", "INFO")
		s.Player.NextActionTick += domain.TimeCostWait
		playerActed = true
	}

	// Если игрок потратил время, запускаем симуляцию мира
	if playerActed {
		s.RunGameLoop()
	}

	// Формируем ответ
	response.World = s.World
	response.Player = s.Player
	response.Entities = s.Entities
	response.Logs = s.Logs

	return response
}

// handlePlayerMove использует System Movement и Combat
func (s *GameService) handlePlayerMove(dx, dy int) bool {
	// 1. Спрашиваем систему движения: "Куда я попаду?"
	res := systems.CalculateMove(s.Player, dx, dy, s.World, s.Entities)

	// 2. Если врезались во врага -> Атака
	if res.BlockedBy != nil && res.BlockedBy.IsHostile {
		logMsg := systems.ApplyAttack(s.Player, res.BlockedBy)
		s.AddLog(logMsg, "COMBAT")
		s.Player.NextActionTick += domain.TimeCostAttackLight
		return true
	}

	// 3. Если путь свободен -> Движение
	if res.HasMoved {
		s.Player.Pos.X = res.NewX
		s.Player.Pos.Y = res.NewY
		s.Player.NextActionTick += domain.TimeCostMove
		return true
	}

	// 4. Если стена
	if res.IsWall {
		s.AddLog("Путь прегражден.", "ERROR")
		return false
	}

	return false
}

// RunGameLoop - Крутит время и ИИ
func (s *GameService) RunGameLoop() {
	loops := 0
	const MaxLoops = 1000

	for {
		// 1. Собираем всех живых
		activeEntities := []*domain.Entity{s.Player}
		for i := range s.Entities {
			if !s.Entities[i].IsDead {
				activeEntities = append(activeEntities, &s.Entities[i])
			}
		}

		// 2. Сортировка по времени (Priority Queue)
		sort.Slice(activeEntities, func(i, j int) bool {
			return activeEntities[i].NextActionTick < activeEntities[j].NextActionTick
		})

		actor := activeEntities[0]
		s.World.GlobalTick = actor.NextActionTick

		// 3. Если ход Игрока -> выход
		if actor.ID == s.Player.ID {
			break
		}

		// 4. Ход NPC
		s.processNPC(actor)

		loops++
		if loops > MaxLoops {
			s.AddLog("System: Time loop break", "ERROR")
			break
		}
	}
}

// processNPC использует System AI
func (s *GameService) processNPC(npc *domain.Entity) {
	// Спрашиваем ИИ: "Что делать?"
	action, target, dx, dy := systems.ComputeNPCAction(npc, s.Player, s.World, s.Entities)

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
