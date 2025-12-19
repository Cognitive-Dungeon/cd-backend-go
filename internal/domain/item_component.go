package domain

import "cognitive-server/internal/core/types"

// ItemComponent описывает предмет в игре.
// Любая Entity с этим компонентом становится предметом.
type ItemComponent struct {
	// Базовые характеристики
	Category    ItemCategory `json:"category"`    // "weapon", "armor", "potion", "food", "misc", "container"
	IsStackable bool         `json:"isStackable"` // можно ли складывать в стаки
	StackSize   int          `json:"stackSize"`   // текущее количество в стаке

	// Характеристики оружия
	Damage      int `json:"damage,omitempty"`      // урон оружия
	AttackSpeed int `json:"attackSpeed,omitempty"` // модификатор скорости атаки (в тиках)

	// Характеристики брони
	Defense   int    `json:"defense,omitempty"`   // защита брони
	ArmorSlot string `json:"armorSlot,omitempty"` // "head", "body", "legs" // TODO: Переделать в enum

	// Характеристики расходуемых предметов (зелья, еда)
	EffectType   string `json:"effectType,omitempty"`   // "heal", "buff_strength", "restore_stamina" // TODO: Переделать в enum
	EffectValue  int    `json:"effectValue,omitempty"`  // величина эффекта
	IsConsumable bool   `json:"isConsumable,omitempty"` // должен ли предмет исчезнуть после использования

	// Физические свойства
	Weight uint `json:"weight"` // вес предмета
	Price  uint `json:"price"`  // стоимость в золоте

	// Взаимодействие
	InteractionRange int  `json:"interactionRange,omitempty"` // радиус взаимодействия (default: 1)
	IsTransparent    bool `json:"isTransparent,omitempty"`    // можно ли смотреть сквозь (для FOV)
	IsIntangible     bool `json:"isIntangible,omitempty"`     // можно ли пройти сквозь

	// Для живых предметов (с личностью)
	IsSentient  bool   `json:"isSentient,omitempty"`  // является ли предмет разумным
	Personality string `json:"personality,omitempty"` // "sadistic", "masochistic", "greedy", "cowardly"
	Chattiness  int    `json:"chattiness,omitempty"`  // частота реплик (0-10)
}

// InventoryComponent хранит предметы у сущности.
type InventoryComponent struct {
	Items         []*Entity `json:"items"`         // ссылки на Entity с ItemComponent
	MaxSlots      int       `json:"maxSlots"`      // максимальное количество слотов
	MaxWeight     uint      `json:"maxWeight"`     // максимальный вес
	CurrentWeight uint      `json:"currentWeight"` // текущий вес
}

// EquipmentComponent хранит экипированные предметы.
type EquipmentComponent struct {
	Weapon *Entity `json:"weapon,omitempty"` // экипированное оружие (Entity с Item.Category="weapon")
	Armor  *Entity `json:"armor,omitempty"`  // экипированная броня (Entity с Item.Category="armor")
	// Можно расширить: Head, Body, Legs, Shield, Ring, Accessory
}

// AddItem добавляет предмет в инвентарь с проверкой места.
func (inv *InventoryComponent) AddItem(item *Entity) bool {
	if inv == nil || item == nil || item.Item == nil {
		return false
	}

	// Проверка на переполнение слотов
	if len(inv.Items) >= inv.MaxSlots {
		return false
	}

	// Проверка веса
	newWeight := inv.CurrentWeight + item.Item.Weight
	if inv.MaxWeight > 0 && newWeight > inv.MaxWeight {
		return false
	}

	// Попытка стакирования
	if item.Item.IsStackable {
		for _, existing := range inv.Items {
			if existing.Item != nil &&
				existing.Name == item.Name &&
				existing.Item.Category == item.Item.Category {
				// Объединяем стаки
				existing.Item.StackSize += item.Item.StackSize
				return true
			}
		}
	}

	// Добавляем как новый слот
	inv.Items = append(inv.Items, item)
	inv.CurrentWeight = newWeight
	return true
}

// RemoveItem удаляет предмет из инвентаря.
func (inv *InventoryComponent) RemoveItem(itemID types.EntityID) *Entity {
	if inv == nil {
		return nil
	}

	for i, item := range inv.Items {
		if item.ID == itemID {
			// Удаляем из слайса
			inv.Items = append(inv.Items[:i], inv.Items[i+1:]...)
			// Обновляем вес
			if item.Item != nil {
				inv.CurrentWeight -= item.Item.Weight
			}
			return item
		}
	}
	return nil
}

// FindItem ищет предмет по ID.
func (inv *InventoryComponent) FindItem(itemID types.EntityID) *Entity {
	if inv == nil {
		return nil
	}

	for _, item := range inv.Items {
		if item.ID == itemID {
			return item
		}
	}
	return nil
}
