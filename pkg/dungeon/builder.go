package dungeon

import (
	"cognitive-server/internal/domain"
	"encoding/json"
	"fmt"
	"math/rand"
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

func (b *LevelBuilder) randRange(min, max int) int {
	return b.rng.Intn(max-min+1) + min
}

// LevelBuilder предоставляет fluent API для создания уровней
type LevelBuilder struct {
	level    int
	width    int
	height   int
	rooms    []Rect
	gameMap  [][]domain.Tile
	entities []domain.Entity
	rng      *rand.Rand
}

// NewLevel создает новый builder для уровня
func NewLevel(level int, rng *rand.Rand) *LevelBuilder {
	return &LevelBuilder{
		level:    level,
		width:    MapWidth,
		height:   MapHeight,
		entities: make([]domain.Entity, 0),
		rng:      rng,
	}
}

// WithSize устанавливает размер карты
func (b *LevelBuilder) WithSize(width, height int) *LevelBuilder {
	b.width = width
	b.height = height
	return b
}

// WithRooms генерирует комнаты и коридоры
func (b *LevelBuilder) WithRooms(maxRooms int) *LevelBuilder {
	// Инициализируем карту стенами
	b.gameMap = make([][]domain.Tile, b.height)
	for y := 0; y < b.height; y++ {
		row := make([]domain.Tile, b.width)
		for x := 0; x < b.width; x++ {
			row[x] = domain.Tile{
				X: x, Y: y, IsWall: true, Env: "stone",
			}
		}
		b.gameMap[y] = row
	}

	// Генерируем комнаты
	b.rooms = make([]Rect, 0, maxRooms)
	for i := 0; i < maxRooms; i++ {
		w := b.randRange(MinSize, MaxSize)
		h := b.randRange(MinSize, MaxSize)
		x := b.randRange(1, b.width-w-1)
		y := b.randRange(1, b.height-h-1)

		newRoom := Rect{X: x, Y: y, W: w, H: h}

		// Проверяем пересечения
		failed := false
		for _, other := range b.rooms {
			if newRoom.Intersects(other) {
				failed = true
				break
			}
		}

		if !failed {
			createRoom(b.gameMap, newRoom)

			// Соединяем с предыдущей комнатой
			if len(b.rooms) > 0 {
				prevX, prevY := b.rooms[len(b.rooms)-1].Center()
				currX, currY := newRoom.Center()

				if b.rng.Intn(2) == 0 {
					createHCorridor(b.gameMap, prevX, currX, prevY)
					createVCorridor(b.gameMap, prevY, currY, currX)
				} else {
					createVCorridor(b.gameMap, prevY, currY, prevX)
					createHCorridor(b.gameMap, prevX, currX, currY)
				}
			}
			b.rooms = append(b.rooms, newRoom)
		}
	}

	return b
}

// SpawnEnemy спавнит врага из шаблона
func (b *LevelBuilder) SpawnEnemy(templateName string, count int) *LevelBuilder {
	template, ok := EnemyTemplates[templateName]
	if !ok {
		return b
	}

	// Спавним в случайных комнатах (кроме первой)
	for i := 0; i < count && len(b.rooms) > 1; i++ {
		roomIdx := b.rng.Intn(len(b.rooms)-1) + 1 // Не в первой комнате
		room := b.rooms[roomIdx]
		cx, cy := room.Center()

		pos := domain.Position{
			X: cx + b.randRange(-1, 1),
			Y: cy + b.randRange(-1, 1),
		}

		// Масштабируем статы по уровню
		scaledTemplate := template
		scaledTemplate.HP += b.level * 2
		scaledTemplate.Strength += b.level / 2

		enemy := scaledTemplate.SpawnEntity(pos, b.level, b.rng)
		b.entities = append(b.entities, enemy)
	}

	return b
}

// SpawnItem спавнит предметы из шаблона
func (b *LevelBuilder) SpawnItem(templateName string, count int) *LevelBuilder {
	template, ok := ItemTemplates[templateName]
	if !ok {
		return b
	}

	// Спавним в случайных комнатах
	for i := 0; i < count && len(b.rooms) > 0; i++ {
		roomIdx := b.rng.Intn(len(b.rooms))
		room := b.rooms[roomIdx]

		// Пробуем найти проходимую клетку (макс 20 попыток)
		var x, y int
		found := false
		for attempt := 0; attempt < 20; attempt++ {
			x = room.X + b.rng.Intn(room.W)
			y = room.Y + b.rng.Intn(room.H)

			// Проверяем, что клетка не стена
			if y >= 0 && y < len(b.gameMap) && x >= 0 && x < len(b.gameMap[y]) {
				if !b.gameMap[y][x].IsWall {
					found = true
					break
				}
			}
		}

		if !found {
			continue // Пропускаем этот предмет, если не нашли место
		}

		pos := domain.Position{X: x, Y: y}
		item := template.SpawnItem(pos, b.level, b.rng)
		b.entities = append(b.entities, *item)
	}

	return b
}

// PlaceExit размещает лестницу
func (b *LevelBuilder) PlaceExit(direction string, targetLevel int) *LevelBuilder {
	if len(b.rooms) == 0 {
		return b
	}

	var room Rect
	var symbol string
	var name string
	var description string

	if direction == "up" {
		room = b.rooms[0] // Первая комната
		symbol = "<"
		name = "Лестница вверх"
		description = "Старая каменная лестница, ведущая на поверхность."
	} else {
		room = b.rooms[len(b.rooms)-1] // Последняя комната
		symbol = ">"
		name = "Лестница вниз"
		description = "Темный проход, ведущий вглубь подземелья."
	}

	cx, cy := room.Center()

	eventPayload, _ := json.Marshal(map[string]interface{}{
		"event":       "LEVEL_TRANSITION",
		"targetLevel": targetLevel,
		"targetPosId": fmt.Sprintf("exit_%s_from_%d", oppositeDirection(direction), targetLevel),
	})

	exit := domain.Entity{
		ID:    fmt.Sprintf("exit_%s_from_%d", direction, b.level),
		Type:  domain.EntityTypeExit,
		Name:  name,
		Pos:   domain.Position{X: cx, Y: cy},
		Level: b.level,
		Render: &domain.RenderComponent{
			Symbol: symbol,
			Color:  "#FFFFFF",
			Label:  symbol,
		},
		Narrative: &domain.NarrativeComponent{
			Description: description,
		},
		Trigger: &domain.TriggerComponent{
			OnInteract: eventPayload,
		},
	}

	b.entities = append(b.entities, exit)
	return b
}

// GetStartPos возвращает стартовую позицию (центр первой комнаты)
func (b *LevelBuilder) GetStartPos() domain.Position {
	if len(b.rooms) > 0 {
		cx, cy := b.rooms[0].Center()
		return domain.Position{X: cx, Y: cy}
	}
	return domain.Position{X: b.width / 2, Y: b.height / 2}
}

// Build собирает и возвращает готовый мир
func (b *LevelBuilder) Build() (*domain.GameWorld, []domain.Entity, domain.Position) {
	world := &domain.GameWorld{
		Map:            b.gameMap,
		Width:          b.width,
		Height:         b.height,
		Level:          b.level,
		SpatialHash:    make(map[int][]*domain.Entity),
		EntityRegistry: make(map[string]*domain.Entity),
	}

	return world, b.entities, b.GetStartPos()
}

// --- Helper functions ---

func oppositeDirection(dir string) string {
	if dir == "up" {
		return "down"
	}
	return "up"
}
