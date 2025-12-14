package dungeon

import (
	"cognitive-server/internal/domain"
	"math/rand"
)

// CreatePlayer generates a new player entity with default starting gear
func CreatePlayer(id string, rng *rand.Rand) *domain.Entity {
	// Создаем героя на основе шаблона
	p := EntityTemplate{
		Name: "Герой " + id[:4], // Берем первые 4 символа ID для краткости
		Type: domain.EntityTypePlayer,
		Render: domain.RenderComponent{
			Symbol: '@',
			Color:  "#22D3EE",
		},
		Narrative: domain.NarrativeComponent{
			Description: "Храбрый исследователь подземелий.",
		},
		Stats: domain.StatsComponent{
			HP:       100,
			Strength: 10,
			Gold:     50,
		},
	}.SpawnEntity(domain.Position{}, 0, rng)

	p.ID = id
	// Инициализируем пустой инвентарь и экипировку
	p.Inventory = &domain.InventoryComponent{Items: []*domain.Entity{}, MaxSlots: 20, MaxWeight: 100}
	p.Equipment = &domain.EquipmentComponent{}

	// Даем стартовое снаряжение
	p.Inventory.AddItem(IronSword.SpawnItem(domain.Position{}, 0, rng))
	p.Inventory.AddItem(HealthPotion.SpawnItem(domain.Position{}, 0, rng))

	return &p
}
