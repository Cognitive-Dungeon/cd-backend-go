package dungeon

import (
	"cognitive-server/internal/domain"
)

// CreatePlayer generates a new player entity with default starting gear
func CreatePlayer(id string) *domain.Entity {
	// Создаем героя на основе шаблона
	p := EntityTemplate{
		Name:        "Герой " + id[:4], // Берем первые 4 символа ID для краткости
		Type:        domain.EntityTypePlayer,
		Symbol:      "@",
		Color:       "#22D3EE",
		Description: "Храбрый исследователь подземелий.",
		HP:          100,
		Strength:    10,
		Gold:        50,
	}.SpawnEntity(domain.Position{}, 0)

	p.ID = id
	// Инициализируем пустой инвентарь и экипировку
	p.Inventory = &domain.InventoryComponent{Items: []*domain.Entity{}, MaxSlots: 20, MaxWeight: 100}
	p.Equipment = &domain.EquipmentComponent{}

	// Даем стартовое снаряжение
	p.Inventory.AddItem(IronSword.SpawnItem(domain.Position{}, 0))
	p.Inventory.AddItem(HealthPotion.SpawnItem(domain.Position{}, 0))

	return &p
}
