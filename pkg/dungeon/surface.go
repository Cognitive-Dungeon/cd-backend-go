package dungeon

import (
	"cognitive-server/internal/domain"
	"fmt"
)

// GenerateSurface создает "домашний" уровень (поверхность).
func GenerateSurface() (*domain.GameWorld, []domain.Entity, domain.Position) {
	world := &domain.GameWorld{
		Map:         make([][]domain.Tile, MapHeight),
		Width:       MapWidth,
		Height:      MapHeight,
		Level:       0, // Поверхность - это уровень 0
		SpatialHash: make(map[int][]*domain.Entity),
		Components:  domain.NewWorldComponents(),
	}
	// ... (здесь можно создать более сложную карту для города)
	// А пока просто сделаем пустую комнату
	for y := 0; y < MapHeight; y++ {
		world.Map[y] = make([]domain.Tile, MapWidth)
		for x := 0; x < MapWidth; x++ {
			isBoundary := x == 0 || y == 0 || x == MapWidth-1 || y == MapHeight-1
			world.Map[y][x] = domain.Tile{X: x, Y: y, IsWall: isBoundary}
		}
	}

	startPos := domain.Position{X: MapWidth / 2, Y: MapHeight / 2}
	var entities []domain.Entity

	// Лестница, ведущая вниз с уровня 0.
	exit := domain.Entity{
		ID:        domain.EntityID(fmt.Sprintf("exit_down_from_%d", 0)),
		Type:      domain.EntityTypeExit,
		Name:      "Спуск в подземелье",
		Pos:       startPos,
		Level:     0,
		Render:    &domain.RenderComponent{Symbol: '>', Color: "#FFFFFF"},
		Narrative: &domain.NarrativeComponent{Description: "Темный проход, ведущий вглубь подземелья."},
	}

	domain.SetComponent(world.Components, exit.ID, domain.TransitionComponent{
		Type:      domain.TransitionTypeStairs,
		Direction: domain.TransitionDirectionDown,
	})

	entities = append(entities, exit)

	return world, entities, startPos
}
