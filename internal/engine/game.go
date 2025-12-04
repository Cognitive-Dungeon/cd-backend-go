package engine

import (
	"cognitive-server/internal/models"
	"cognitive-server/pkg/dungeon"
	"encoding/json"
	"fmt"
	"time"
)

type GameEngine struct {
	World    *models.GameWorld
	Player   *models.Entity
	Entities []models.Entity
	Logs     []models.LogEntry
}

func NewGame() *GameEngine {
	// Генерируем Уровень 1
	world, entities, startPos := dungeon.Generate(1)

	// Создаем игрока
	player := &models.Entity{
		ID:     "p1",
		Label:  "Hero",
		Name:   "Герой",
		Symbol: "@",
		Color:  "text-cyan-400",
		Type:   models.EntityTypePlayer,
		Pos:    startPos,
		Stats: models.Stats{
			HP: 100, MaxHP: 100, Stamina: 100, MaxStamina: 100, Gold: 50, Strength: 10,
		},
	}

	return &GameEngine{
		World:    world,
		Player:   player,
		Entities: entities,
		Logs:     []models.LogEntry{},
	}
}

// ProcessCommand - главный метод обработки ввода
func (g *GameEngine) ProcessCommand(cmd models.ClientCommand) *models.ServerResponse {
	// Очищаем логи перед новым ответом
	g.Logs = []models.LogEntry{}

	response := &models.ServerResponse{Type: "UPDATE"}

	switch cmd.Action {
	case "INIT":
		g.AddLog("Добро пожаловать в Cognitive Dungeon (Go Server).", "INFO")
		response.Type = "INIT"

	case "MOVE":
		var p models.MovePayload
		if err := json.Unmarshal(cmd.Payload, &p); err == nil {
			g.handleMove(p.Dx, p.Dy)
		}

	case "WAIT":
		g.AddLog("Вы ждете...", "INFO")
		g.World.GlobalTick += 50
	}

	// Собираем снапшот
	response.World = g.World
	response.Player = g.Player
	response.Entities = g.Entities
	response.Logs = g.Logs

	return response
}

func (g *GameEngine) handleMove(dx, dy int) {
	newX := g.Player.Pos.X + dx
	newY := g.Player.Pos.Y + dy

	// Проверка границ и стен
	if newX < 0 || newX >= g.World.Width || newY < 0 || newY >= g.World.Height {
		return
	}
	if g.World.Map[newY][newX].IsWall {
		g.AddLog("Путь прегражден.", "ERROR")
		return
	}

	// Обновляем позицию
	g.Player.Pos.X = newX
	g.Player.Pos.Y = newY
	g.AddLog(fmt.Sprintf("Вы переместились в %d,%d", newX, newY), "INFO")

	// Имитация траты времени
	g.World.GlobalTick += 100
}

func (g *GameEngine) AddLog(text, logType string) {
	g.Logs = append(g.Logs, models.LogEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:      text,
		Type:      logType,
		Timestamp: time.Now().UnixMilli(),
	})
}
