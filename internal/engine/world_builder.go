package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/dungeon"
	"math/rand"
)

// buildInitialWorld создает все начальные уровни, сущности и игрока.
func buildInitialWorld(masterSeed int64) (map[int]*domain.GameWorld, []*domain.Entity, map[int]int64) {
	levelSeeds := make(map[int]int64)
	worlds := make(map[int]*domain.GameWorld)
	var allEntities []*domain.Entity

	// --- УРОВЕНЬ 0 (Surface) ---
	// Сид для уровня 0 всегда равен MasterSeed
	seed0 := masterSeed
	levelSeeds[0] = seed0

	// Surface пока процедурно не генерируется, но на будущее seed готов
	surfaceWorld, surfaceEntities, startPos := dungeon.GenerateSurface()
	worlds[0] = surfaceWorld

	// --- УРОВЕНЬ 1 (Dungeon) ---
	// Детерминированная деривация сида: Master + LevelID.
	// Можно использовать хеширование для лучшего разброса, но сложение для старта ок.
	seed1 := masterSeed + 1
	levelSeeds[1] = seed1

	// Создаем изолированный RNG для генерации этого уровня
	rng1 := rand.New(rand.NewSource(seed1))

	dungeonWorld1, dungeonEntities1, _ := dungeon.NewLevel(1, rng1).
		WithRooms(8).
		SpawnEnemy("goblin", 3).
		SpawnEnemy("orc", 1).
		SpawnItem("health_potion", 2).
		SpawnItem("bread", 3).
		SpawnItem("leather_armor", 1).
		SpawnItem("steel_dagger", 1).
		PlaceExit("up", 0).
		PlaceExit("down", 2).
		Build()
	worlds[1] = dungeonWorld1

	// --- ИГРОК ---
	player := dungeon.CreatePlayer("hero_1")
	player.Pos = startPos
	player.Level = 0
	allEntities = append(allEntities, player)

	// Собираем сущностей
	for i := range surfaceEntities {
		allEntities = append(allEntities, &surfaceEntities[i])
	}
	for i := range dungeonEntities1 {
		allEntities = append(allEntities, &dungeonEntities1[i])
	}

	// Регистрируем
	for _, e := range allEntities {
		if world, ok := worlds[e.Level]; ok {
			world.RegisterEntity(e)
			world.AddEntity(e)
		}
	}

	return worlds, allEntities, levelSeeds
}
