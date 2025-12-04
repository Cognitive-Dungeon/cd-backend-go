package engine

import (
	"cognitive-server/internal/models"
	"cognitive-server/pkg/dungeon"
	"encoding/json"
	"fmt"
	"math"
	"sort"
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
	g.Logs = []models.LogEntry{}
	response := &models.ServerResponse{Type: "UPDATE"}

	playerActed := false // Флаг: совершил ли игрок действие, требующее времени

	switch cmd.Action {
	case "INIT":
		g.AddLog("Добро пожаловать в Cognitive Dungeon.", "INFO")
		response.Type = "INIT"

	case "MOVE":
		var p models.MovePayload
		if err := json.Unmarshal(cmd.Payload, &p); err == nil {
			// Если handleMove вернул true, значит ход сделан
			if g.handleMove(p.Dx, p.Dy) {
				playerActed = true
			}
		}

	case "WAIT":
		g.AddLog("Вы пропускаете ход.", "INFO")
		g.Player.NextActionTick += models.TimeCostWait
		playerActed = true
	}

	// Если игрок что-то сделал, запускаем мир
	if playerActed {
		g.RunGameLoop()
	}

	// Сборка ответа...
	response.World = g.World
	response.Player = g.Player
	response.Entities = g.Entities
	response.Logs = g.Logs

	return response
}

// Модифицированный handleMove возвращает bool (успех)
func (g *GameEngine) handleMove(dx, dy int) bool {
	newX := g.Player.Pos.X + dx
	newY := g.Player.Pos.Y + dy

	if g.isBlocked(newX, newY) {
		// Проверяем, может это атака?
		for i := range g.Entities {
			e := &g.Entities[i]
			if !e.IsDead && e.Pos.X == newX && e.Pos.Y == newY && e.IsHostile {
				// АТАКА
				damage := g.Player.Stats.Strength
				e.Stats.HP -= damage
				g.AddLog(fmt.Sprintf("Вы ударили %s на %d урона.", e.Name, damage), "COMBAT")
				if e.Stats.HP <= 0 {
					e.IsDead = true
					e.Symbol = "%" // Труп
					e.Color = "text-gray-500"
					g.AddLog(fmt.Sprintf("%s умирает.", e.Name), "COMBAT")
				}

				g.Player.NextActionTick += models.TimeCostAttackLight
				return true
			}
		}

		g.AddLog("Путь прегражден.", "ERROR")
		return false
	}

	g.Player.Pos.X = newX
	g.Player.Pos.Y = newY

	// Теперь время добавляется к NextActionTick, а не глобальному
	g.Player.NextActionTick += models.TimeCostMove
	return true
}

func (g *GameEngine) AddLog(text, logType string) {
	g.Logs = append(g.Logs, models.LogEntry{
		ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
		Text:      text,
		Type:      logType,
		Timestamp: time.Now().UnixMilli(),
	})
}

// RunGameLoop - Главный цикл времени
func (g *GameEngine) RunGameLoop() {
	// Предохранитель от бесконечного цикла (если NPC тупят)
	const MaxLoops = 1000
	loops := 0

	for {
		// 1. Собираем всех активных участников (Игрок + Живые NPC)
		var activeEntities []*models.Entity
		activeEntities = append(activeEntities, g.Player)

		for i := range g.Entities {
			if !g.Entities[i].IsDead {
				activeEntities = append(activeEntities, &g.Entities[i])
			}
		}

		// 2. Сортируем их по времени (кто раньше освободится, тот и ходит)
		sort.Slice(activeEntities, func(i, j int) bool {
			return activeEntities[i].NextActionTick < activeEntities[j].NextActionTick
		})

		// Кто сейчас ходит?
		currentActor := activeEntities[0]

		// 3. Обновляем глобальное время
		g.World.GlobalTick = currentActor.NextActionTick

		// 4. Если ходит Игрок -> Стоп машина, ждем ввода
		if currentActor.ID == g.Player.ID {
			break
		}

		// 5. Если ходит NPC -> Выполняем его действие
		g.processNPC(currentActor)

		// Защита от зависания
		loops++
		if loops > MaxLoops {
			g.AddLog("System: Time Loop break triggered", "ERROR")
			break
		}
	}
}

// processNPC - Хардкорная логика (пока без LLM)
func (g *GameEngine) processNPC(npc *models.Entity) {
	// Если NPC враждебен и видит игрока
	dist := distance(npc.Pos, g.Player.Pos)

	if npc.IsHostile && dist <= models.AggroRadius {
		if dist <= 1.5 { // Рядом (диагональ считается за 1.41)
			// АТАКА
			damage := npc.Stats.Strength
			g.Player.Stats.HP -= damage
			g.AddLog(fmt.Sprintf("%s бьет вас! -%d HP", npc.Name, damage), "COMBAT")

			// Тратим время NPC
			npc.NextActionTick += models.TimeCostAttackLight
		} else {
			// ДВИЖЕНИЕ К ИГРОКУ (Smart Sliding)
			dx := g.Player.Pos.X - npc.Pos.X
			dy := g.Player.Pos.Y - npc.Pos.Y

			stepX := sign(dx)
			stepY := sign(dy)

			// 1. Попытка идеального хода (по диагонали или прямой)
			nextX, nextY := npc.Pos.X+stepX, npc.Pos.Y+stepY

			// Если идеальный путь заблокирован, пробуем альтернативы
			if g.isBlocked(nextX, nextY) {
				// Определяем приоритетную ось (где расстояние больше)
				tryXFirst := math.Abs(float64(dx)) > math.Abs(float64(dy))

				moved := false

				if tryXFirst {
					// Пробуем только X
					if stepX != 0 && !g.isBlocked(npc.Pos.X+stepX, npc.Pos.Y) {
						nextX, nextY = npc.Pos.X+stepX, npc.Pos.Y
						moved = true
					} else if stepY != 0 && !g.isBlocked(npc.Pos.X, npc.Pos.Y+stepY) {
						// Если X занят, пробуем Y
						nextX, nextY = npc.Pos.X, npc.Pos.Y+stepY
						moved = true
					}
				} else {
					// Пробуем только Y
					if stepY != 0 && !g.isBlocked(npc.Pos.X, npc.Pos.Y+stepY) {
						nextX, nextY = npc.Pos.X, npc.Pos.Y+stepY
						moved = true
					} else if stepX != 0 && !g.isBlocked(npc.Pos.X+stepX, npc.Pos.Y) {
						// Если Y занят, пробуем X
						nextX, nextY = npc.Pos.X+stepX, npc.Pos.Y
						moved = true
					}
				}

				if !moved {
					// Враг в тупике или зажат
					npc.NextActionTick += models.TimeCostWait
					return
				}
			}

			// Применяем движение
			npc.Pos.X = nextX
			npc.Pos.Y = nextY
			npc.NextActionTick += models.TimeCostMove
		}
	} else {
		// IDLE: Просто стоит и ждет
		npc.NextActionTick += models.TimeCostWait + 50
	}
}

// Вспомогательные функции
func distance(p1, p2 models.Position) float64 {
	return math.Sqrt(math.Pow(float64(p1.X-p2.X), 2) + math.Pow(float64(p1.Y-p2.Y), 2))
}

func sign(x int) int {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

func (g *GameEngine) isBlocked(x, y int) bool {
	// Границы карты
	if x < 0 || x >= g.World.Width || y < 0 || y >= g.World.Height {
		return true
	}
	// Стены
	if g.World.Map[y][x].IsWall {
		return true
	}
	// Игрок
	if x == g.Player.Pos.X && y == g.Player.Pos.Y {
		return true
	}
	// Другие NPC
	for _, e := range g.Entities {
		if !e.IsDead && e.Pos.X == x && e.Pos.Y == y {
			return true
		}
	}
	return false
}
