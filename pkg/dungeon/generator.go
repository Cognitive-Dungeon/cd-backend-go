package dungeon

import (
	"cognitive-server/internal/domain"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// Константы генерации
const (
	MapWidth  = 40
	MapHeight = 25
	MaxRooms  = 8
	MinSize   = 4
	MaxSize   = 10
)

// Rect - Вспомогательная структура для комнаты
type Rect struct {
	X, Y, W, H int
}

func (r Rect) Center() (int, int) {
	return r.X + r.W/2, r.Y + r.H/2
}

func (r Rect) Intersects(other Rect) bool {
	return r.X <= other.X+other.W && r.X+r.W >= other.X &&
		r.Y <= other.Y+other.H && r.Y+r.H >= other.Y
}

// Список ключей предметов для случайного выбора
var commonItems = []string{
	"health_potion", "gold", "torch", "bread", "meat",
	"iron_sword", "leather_armor", "wooden_club",
}

// Generate создает новый уровень
func Generate(level int, r *rand.Rand) (*domain.GameWorld, []domain.Entity, domain.Position) {
	rand.Seed(time.Now().UnixNano())

	// 1. Заполняем стенами
	gameMap := make([][]domain.Tile, MapHeight)
	for y := 0; y < MapHeight; y++ {
		row := make([]domain.Tile, MapWidth)
		for x := 0; x < MapWidth; x++ {
			row[x] = domain.Tile{X: x, Y: y, IsWall: true, Env: "stone"}
		}
		gameMap[y] = row
	}

	var rooms []Rect
	var entities []domain.Entity

	// 2. Генерируем комнаты
	for i := 0; i < MaxRooms; i++ {
		w := randRange(MinSize, MaxSize)
		h := randRange(MinSize, MaxSize)
		x := randRange(1, MapWidth-w-1)
		y := randRange(1, MapHeight-h-1)

		newRoom := Rect{X: x, Y: y, W: w, H: h}
		failed := false

		for _, other := range rooms {
			if newRoom.Intersects(other) {
				failed = true
				break
			}
		}

		if !failed {
			createRoom(gameMap, newRoom)
			if len(rooms) > 0 {
				prevX, prevY := rooms[len(rooms)-1].Center()
				currX, currY := newRoom.Center()
				if rand.Intn(2) == 0 {
					createHCorridor(gameMap, prevX, currX, prevY)
					createVCorridor(gameMap, prevY, currY, currX)
				} else {
					createVCorridor(gameMap, prevY, currY, prevX)
					createHCorridor(gameMap, prevX, currX, currY)
				}
			}
			rooms = append(rooms, newRoom)
		}
	}

	// 3. Спавн игрока (в первой комнате)
	startPos := domain.Position{X: 0, Y: 0}
	if len(rooms) > 0 {
		cx, cy := rooms[0].Center()
		startPos = domain.Position{X: cx, Y: cy}

		// Лестница ВВЕРХ
		eventUp, _ := json.Marshal(map[string]interface{}{
			"event":       "LEVEL_TRANSITION",
			"targetLevel": level - 1,
			"targetPosId": fmt.Sprintf("exit_down_from_%d", level-1),
		})
		entities = append(entities, domain.Entity{
			ID:      fmt.Sprintf("exit_up_from_%d", level),
			Type:    domain.EntityTypeExit,
			Name:    "Лестница вверх",
			Pos:     domain.Position{X: cx, Y: cy},
			Level:   level,
			Render:  &domain.RenderComponent{Symbol: "<", Color: "#FFFFFF", Label: "<"},
			Trigger: &domain.TriggerComponent{OnInteract: eventUp},
		})
	}

	// 4. Заполнение комнат (Враги И Предметы)
	for i := 1; i < len(rooms); i++ {
		room := rooms[i]
		cx, cy := room.Center()

		// --- ВРАГИ (30% шанс) ---
		if r.Float32() > 0.7 {
			isOrc := r.Float32() > 0.7 || level > 3
			enemy := domain.Entity{
				ID:    domain.GenerateID(),
				Type:  domain.EntityTypeEnemy,
				Pos:   domain.Position{X: cx + randRange(-1, 1), Y: cy + randRange(-1, 1)},
				Level: level,
				Stats: &domain.StatsComponent{
					HP: 15 + level*2, MaxHP: 15 + level*2, Strength: 2 + level/2, Gold: randRange(1, 10),
				},
				AI:     &domain.AIComponent{IsHostile: true, State: "IDLE"},
				Render: &domain.RenderComponent{},
				Vision: &domain.VisionComponent{Radius: domain.VisionRadius},
				Memory: &domain.MemoryComponent{ExploredPerLevel: make(map[int]map[int]bool)},
			}
			if isOrc {
				enemy.Name = "Свирепый Орк"
				enemy.Render.Symbol = "O"
				enemy.Render.Color = "#DC2626"
				enemy.Stats.HP *= 2
				enemy.Stats.MaxHP = enemy.Stats.HP
			} else {
				enemy.Name = "Хитрый Гоблин"
				enemy.Render.Symbol = "g"
				enemy.Render.Color = "#22C55E"
			}
			entities = append(entities, enemy)
		}

		// --- ПРЕДМЕТЫ (50% шанс) ---
		// Спавним 1-2 предмета в комнате
		if r.Float32() > 0.5 {
			count := randRange(1, 2)
			for j := 0; j < count; j++ {
				// Выбираем случайный шаблон
				itemKey := commonItems[r.Intn(len(commonItems))]
				template := ItemTemplates[itemKey]

				// Случайная позиция внутри комнаты
				ix := room.X + 1 + r.Intn(room.W-2)
				iy := room.Y + 1 + r.Intn(room.H-2)

				itemEnt := template.SpawnItem(domain.Position{X: ix, Y: iy}, level)
				entities = append(entities, *itemEnt)
			}
		}
	}

	// 5. Лестница ВНИЗ (в последней комнате)
	if len(rooms) > 0 {
		lx, ly := rooms[len(rooms)-1].Center()
		eventDown, _ := json.Marshal(map[string]interface{}{
			"event":       "LEVEL_TRANSITION",
			"targetLevel": level + 1,
			"targetPosId": fmt.Sprintf("exit_up_from_%d", level+1),
		})
		entities = append(entities, domain.Entity{
			ID:      fmt.Sprintf("exit_down_from_%d", level),
			Type:    domain.EntityTypeExit,
			Name:    "Лестница вниз",
			Pos:     domain.Position{X: lx, Y: ly},
			Level:   level,
			Render:  &domain.RenderComponent{Symbol: ">", Color: "#FFFFFF", Label: ">"},
			Trigger: &domain.TriggerComponent{OnInteract: eventDown},
		})
	}

	return &domain.GameWorld{
		Map:         gameMap,
		Width:       MapWidth,
		Height:      MapHeight,
		Level:       level,
		SpatialHash: make(map[int][]*domain.Entity),
	}, entities, startPos
}

// --- Вспомогательные функции ---

func createRoom(gameMap [][]domain.Tile, room Rect) {
	for y := room.Y + 1; y < room.Y+room.H; y++ {
		for x := room.X + 1; x < room.X+room.W; x++ {
			gameMap[y][x].IsWall = false
			gameMap[y][x].Env = "floor"
		}
	}
}

func createHCorridor(gameMap [][]domain.Tile, x1, x2, y int) {
	start := min(x1, x2)
	end := max(x1, x2)
	for x := start; x <= end; x++ {
		gameMap[y][x].IsWall = false
		gameMap[y][x].Env = "floor"
	}
}

func createVCorridor(gameMap [][]domain.Tile, y1, y2, x int) {
	start := min(y1, y2)
	end := max(y1, y2)
	for y := start; y <= end; y++ {
		gameMap[y][x].IsWall = false
		gameMap[y][x].Env = "floor"
	}
}

func randRange(min, max int) int {
	return rand.Intn(max-min+1) + min
}
