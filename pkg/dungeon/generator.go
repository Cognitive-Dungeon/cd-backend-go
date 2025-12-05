package dungeon

import (
	"cognitive-server/internal/domain"
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

// Generate создает новый уровень
func Generate(level int) (*domain.GameWorld, []domain.Entity, domain.Position) {
	// Инициализируем рандом (важно, иначе карта будет одинаковой)
	// В Go 1.20+ глобальный сид рандомен, но для надежности можно так:
	rand.Seed(time.Now().UnixNano())

	// 1. Заполняем стенами
	gameMap := make([][]domain.Tile, MapHeight)
	for y := 0; y < MapHeight; y++ {
		row := make([]domain.Tile, MapWidth)
		for x := 0; x < MapWidth; x++ {
			row[x] = domain.Tile{
				X: x, Y: y, IsWall: true, Env: "stone",
				IsVisible: false, IsExplored: false,
			}
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
				// Соединяем с предыдущей комнатой
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

	// 3. Спавн игрока (в центре первой комнаты)
	startPos := domain.Position{X: 0, Y: 0}
	if len(rooms) > 0 {
		cx, cy := rooms[0].Center()
		startPos = domain.Position{X: cx, Y: cy}

		// Лестница ВВЕРХ там же
		entities = append(entities, domain.Entity{
			ID: "exit_up", Label: "<", Symbol: "<", Type: domain.EntityTypeExit, Color: "text-white",
			Name: "Лестница вверх", Pos: domain.Position{X: cx, Y: cy},
		})
	}

	// 4. Спавн врагов и предметов (во всех комнатах кроме первой)
	for i := 1; i < len(rooms); i++ {
		room := rooms[i]
		cx, cy := room.Center()

		// Шанс спавна врага
		if rand.Float32() > 0.3 {
			isOrc := rand.Float32() > 0.7 || level > 3

			enemy := domain.Entity{
				ID:        fmt.Sprintf("e_%d", i),
				Type:      domain.EntityTypeEnemy,
				IsHostile: true,
				Stats: domain.Stats{
					HP: 15 + level*2, MaxHP: 15 + level*2,
					Strength: 2 + level/2,
				},
				NextActionTick: 0,
			}

			if isOrc {
				enemy.Name = "Свирепый Орк"
				enemy.Symbol = "O"
				enemy.Color = "text-red-600"
				enemy.Personality = "Furious"
				enemy.Stats.HP *= 2
				enemy.Stats.Strength += 2
			} else {
				enemy.Name = "Хитрый Гоблин"
				enemy.Symbol = "g"
				enemy.Color = "text-green-500"
				enemy.Personality = "Cowardly"
			}

			// Небольшой сдвиг от центра комнаты, чтобы не стоять друг на друге
			enemy.Pos = domain.Position{X: cx + randRange(-1, 1), Y: cy + randRange(-1, 1)}
			entities = append(entities, enemy)
		}
	}

	// 5. Лестница ВНИЗ (в последней комнате)
	if len(rooms) > 0 {
		lastRoom := rooms[len(rooms)-1]
		lx, ly := lastRoom.Center()
		entities = append(entities, domain.Entity{
			ID: "exit_down", Label: ">", Symbol: ">", Type: domain.EntityTypeExit, Color: "text-white",
			Name: "Лестница вниз", Pos: domain.Position{X: lx, Y: ly},
		})
	}

	world := &domain.GameWorld{
		Map:        gameMap,
		Width:      MapWidth,
		Height:     MapHeight,
		Level:      level,
		GlobalTick: 0,
	}

	return world, entities, startPos
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
