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

	CommandChan chan domain.ClientCommand
}

func NewService() *GameService {
	// Генерация (без изменений)
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
		CommandChan: make(chan domain.ClientCommand, 100), // Буфер побольше
	}
}

// Start - запускает "сердце" сервера
func (s *GameService) Start() {
	go s.RunGameLoop()
}

// ProcessCommand - Публичный метод: просто кладет в очередь (не блокирует)
func (s *GameService) ProcessCommand(cmd domain.ClientCommand) {
	select {
	case s.CommandChan <- cmd:
	default:
		fmt.Println("Command queue full")
	}
}

// GetState возвращает снапшот и очищает очередь логов,
// чтобы они не дублировались при следующем обновлении.
func (s *GameService) GetState() *domain.ServerResponse {
	// 1. Копируем текущие логи
	currentLogs := make([]domain.LogEntry, len(s.Logs))
	copy(currentLogs, s.Logs)

	// 2. Очищаем массив в сервисе
	s.Logs = []domain.LogEntry{}

	// 3. Возвращаем копию
	return &domain.ServerResponse{
		Type:     "UPDATE",
		World:    s.World,
		Player:   s.Player,
		Entities: s.Entities,
		Logs:     currentLogs, // Отдаем только новые
	}
}

// --- ГЛАВНЫЙ ЦИКЛ АРБИТРА ---

func (s *GameService) RunGameLoop() {
	for {
		// 1. Определяем, чей сейчас ход (на основе Времени)
		activeActor := s.getNextActor()

		// 2. Обновляем глобальное время
		s.World.GlobalTick = activeActor.NextActionTick

		// 3. Если ход ИГРОКА
		if activeActor.ID == s.Player.ID {
			// Мы БЛОКИРУЕМ цикл и ждем команду от клиента.
			// Потому что пока игрок не походит, время в мире стопнуто.
			select {
			case cmd := <-s.CommandChan:
				s.executeCommand(cmd) // Выполняем команду
			}
		} else {
			// 4. Если ход NPC
			// Проверяем, не пришла ли системная команда (например, выход игрока), пока ходит NPC
			select {
			case cmd := <-s.CommandChan:
				// Обрабатываем приоритетные команды даже в ход NPC (опционально)
				if cmd.Action == "INIT" {
					s.executeCommand(cmd)
				}
			default:
				// Если команд нет - NPC делает ход
				s.processNPC(activeActor)
			}
		}
	}
}

// executeCommand - Внутренняя логика обработки (бывший ProcessCommand)
func (s *GameService) executeCommand(cmd domain.ClientCommand) {
	// Очищаем логи (или накапливаем, тут зависит от геймдизайна)
	// s.Logs = []domain.LogEntry{}

	switch cmd.Action {
	case "INIT":
		s.AddLog("Добро пожаловать в Cognitive Dungeon.", "INFO")

	case "MOVE":
		var p domain.MovePayload
		if err := json.Unmarshal(cmd.Payload, &p); err == nil {
			s.handlePlayerMove(p.Dx, p.Dy)
		}

	case "WAIT":
		s.AddLog("Вы ждете...", "INFO")
		s.Player.NextActionTick += domain.TimeCostWait
	}
}

// --- УТИЛИТЫ ---

func (s *GameService) getNextActor() *domain.Entity {
	// Собираем всех живых
	activeEntities := []*domain.Entity{s.Player}
	for i := range s.Entities {
		if !s.Entities[i].IsDead {
			activeEntities = append(activeEntities, &s.Entities[i])
		}
	}
	// Сортируем: кто меньше ждал, тот и ходит
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
}

func (s *GameService) processNPC(npc *domain.Entity) {
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
