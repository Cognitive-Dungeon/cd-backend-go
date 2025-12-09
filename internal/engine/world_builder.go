package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/dungeon"
	"math/rand"
	"time"
)

// buildInitialWorld создает все начальные уровни, сущности и игрока.
func buildInitialWorld() (map[int]*domain.GameWorld, []*domain.Entity) {
	worlds := make(map[int]*domain.GameWorld)
	var allEntities []*domain.Entity

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// 1. Генерируем миры
	surfaceWorld, surfaceEntities, startPos := dungeon.GenerateSurface()
	worlds[0] = surfaceWorld

	dungeonWorld1, dungeonEntities1, _ := dungeon.Generate(1, rng)
	worlds[1] = dungeonWorld1

	// 2. Создаем игрока
	player := &domain.Entity{
		ID:     "hero_1",
		Name:   "Герой",
		Type:   domain.EntityTypePlayer,
		Pos:    startPos,
		Level:  0,
		Render: &domain.RenderComponent{Symbol: "@", Color: "#22D3EE", Label: "A"},
		Stats: &domain.StatsComponent{
			HP: 100, MaxHP: 100, Stamina: 100, MaxStamina: 100, Gold: 50, Strength: 10,
		},
		AI:        &domain.AIComponent{NextActionTick: 0, IsHostile: false},
		Narrative: &domain.NarrativeComponent{Description: "Искатель приключений."},
		Vision:    &domain.VisionComponent{Radius: domain.VisionRadius},
		Memory:    &domain.MemoryComponent{ExploredPerLevel: make(map[int]map[int]bool)},
	}
	allEntities = append(allEntities, player)

	// 3. Собираем всех сущностей в один список
	for i := range surfaceEntities {
		allEntities = append(allEntities, &surfaceEntities[i])
	}
	for i := range dungeonEntities1 {
		allEntities = append(allEntities, &dungeonEntities1[i])
	}

	// 4. Регистрируем сущностей и размещаем их в мирах
	for _, e := range allEntities {
		if world, ok := worlds[e.Level]; ok {
			world.RegisterEntity(e)
			world.AddEntity(e)
		}
	}

	return worlds, allEntities
}
