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

	// 1. Генерируем миры с помощью нового API
	surfaceWorld, surfaceEntities, startPos := dungeon.GenerateSurface()
	worlds[0] = surfaceWorld

	// Уровень 1: Начальное подземелье (гоблины + немного орков + предметы)
	dungeonWorld1, dungeonEntities1, _ := dungeon.NewLevel(1, rng).
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
		Inventory: &domain.InventoryComponent{
			Items:     []*domain.Entity{},
			MaxSlots:  20,
			MaxWeight: 100,
		},
		Equipment: &domain.EquipmentComponent{},
	}

	// Даём игроку стартовые предметы
	startingSword := dungeon.IronSword.SpawnItem(startPos, 0)
	startingPotion := dungeon.HealthPotion.SpawnItem(startPos, 0)

	player.Inventory.AddItem(startingSword)
	player.Inventory.AddItem(startingPotion)

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
