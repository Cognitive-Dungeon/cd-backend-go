package systems

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/utils"
	"fmt"
)

// --- PICKUP ---

func TryPickup(actor *domain.Entity, item *domain.Entity, world *domain.GameWorld) (string, error) {
	if actor.Inventory == nil {
		return "", fmt.Errorf("%s не может иметь инвентарь", actor.Name)
	}
	if item.Item == nil {
		return "", fmt.Errorf("это не предмет")
	}

	if !actor.Inventory.AddItem(item) {
		return "", fmt.Errorf("инвентарь полон или перегруз")
	}

	// Удаляем из мира (используем наш новый безопасный метод)
	world.RemoveEntity(item)
	item.Level = -1 // Убираем в "лимбо"

	return fmt.Sprintf("%s подбирает %s.", actor.Name, item.Name), nil
}

// --- DROP ---

func TryDrop(actor *domain.Entity, itemID domain.EntityID, count int, world *domain.GameWorld) (string, error) {
	if actor.Inventory == nil {
		return "", fmt.Errorf("нет инвентаря")
	}

	item := actor.Inventory.FindItem(itemID)
	if item == nil {
		return "", fmt.Errorf("предмет не найден")
	}

	// Обработка стаков (если просят выбросить часть)
	if count > 0 && item.Item.IsStackable && item.Item.StackSize > count {
		// Уменьшаем стак в руках
		item.Item.StackSize -= count

		// Создаем копию для выброса
		droppedItem := *item // Shallow copy структуры Entity
		// Глубокая копия компонентов, которые меняются
		droppedItem.ID = utils.GenerateID()
		droppedItem.Item = &domain.ItemComponent{}
		*droppedItem.Item = *item.Item // Copy Item data
		droppedItem.Item.StackSize = count

		placeOnGround(actor, &droppedItem, world)
		return fmt.Sprintf("%s выбрасывает %dx %s.", actor.Name, count, item.Name), nil
	}

	// Выбрасываем целиком
	actor.Inventory.RemoveItem(itemID)
	placeOnGround(actor, item, world)

	return fmt.Sprintf("%s выбрасывает %s.", actor.Name, item.Name), nil
}

func placeOnGround(actor *domain.Entity, item *domain.Entity, world *domain.GameWorld) {
	item.Pos = actor.Pos
	item.Level = actor.Level
	world.RegisterEntity(item)
	world.AddEntity(item)
}

// --- EQUIP ---

func TryEquip(actor *domain.Entity, itemID domain.EntityID) (string, error) {
	if actor.Inventory == nil || actor.Equipment == nil {
		return "", fmt.Errorf("невозможно экипировать")
	}

	item := actor.Inventory.FindItem(itemID)
	if item == nil {
		return "", fmt.Errorf("предмет не найден")
	}
	if item.Item == nil {
		return "", fmt.Errorf("это не экипировка")
	}

	var oldItem *domain.Entity

	switch item.Item.Category {
	case domain.ItemCategoryWeapon:
		oldItem = actor.Equipment.Weapon
		actor.Equipment.Weapon = item
	case domain.ItemCategoryArmor:
		oldItem = actor.Equipment.Armor
		actor.Equipment.Armor = item
	default:
		return "", fmt.Errorf("этот предмет нельзя надеть")
	}

	// Если что-то было надето — снимаем это (возвращаем в слот инвентаря, оно там и так есть, просто перестает быть ссылкой в Equipment)
	// Важно: В данной архитектуре предмет, который экипирован, ВСЕ ЕЩЕ находится в списке Inventory.Items.
	// EquipComponent просто хранит ссылку на него. Поэтому "снимать" в инвентарь не нужно, оно уже там.
	// НО! Если вы хотите, чтобы надетые вещи исчезали из рюкзака — логика будет другой.
	// Сейчас предполагаем, что `Inventory` это "все вещи персонажа", а `Equipment` это "что из них активно".

	msg := fmt.Sprintf("%s экипирует %s.", actor.Name, item.Name)
	if oldItem != nil {
		msg = fmt.Sprintf("%s снимает %s и берет %s.", actor.Name, oldItem.Name, item.Name)
	}
	return msg, nil
}

// --- UNEQUIP ---

func TryUnequip(actor *domain.Entity, itemID domain.EntityID) (string, error) {
	if actor.Equipment == nil {
		return "", fmt.Errorf("нет слотов экипировки")
	}

	var itemName string

	if actor.Equipment.Weapon != nil && actor.Equipment.Weapon.ID == itemID {
		itemName = actor.Equipment.Weapon.Name
		actor.Equipment.Weapon = nil
	} else if actor.Equipment.Armor != nil && actor.Equipment.Armor.ID == itemID {
		itemName = actor.Equipment.Armor.Name
		actor.Equipment.Armor = nil
	} else {
		return "", fmt.Errorf("этот предмет не надет")
	}

	return fmt.Sprintf("%s снимает %s.", actor.Name, itemName), nil
}

// --- USE (Consumables) ---

func TryUse(actor *domain.Entity, itemID domain.EntityID) (string, error) {
	if actor.Inventory == nil {
		return "", fmt.Errorf("нет инвентаря")
	}

	item := actor.Inventory.FindItem(itemID)
	if item == nil {
		return "", fmt.Errorf("предмет не найден")
	}
	if item.Item == nil || item.Item.EffectType == "" {
		return "", fmt.Errorf("предмет нельзя использовать")
	}

	// Применение эффектов
	effectMsg := applyEffect(actor, item.Item)
	if effectMsg == "" {
		return "", fmt.Errorf("эффект не сработал")
	}

	// Расходование
	if item.Item.IsConsumable {
		if item.Item.IsStackable && item.Item.StackSize > 1 {
			item.Item.StackSize--
		} else {
			actor.Inventory.RemoveItem(itemID)
		}
	}

	return effectMsg, nil
}

func applyEffect(actor *domain.Entity, props *domain.ItemComponent) string {
	if actor.Stats == nil {
		return ""
	}

	switch props.EffectType {
	case "heal":
		actor.Stats.Heal(props.EffectValue)
		return fmt.Sprintf("%s лечится на %d HP.", actor.Name, props.EffectValue)
	case "restore_stamina":
		actor.Stats.RestoreStamina(props.EffectValue)
		return fmt.Sprintf("%s восстанавливает %d сил.", actor.Name, props.EffectValue)
	case "buff_strength":
		actor.Stats.Strength += props.EffectValue
		return fmt.Sprintf("%s чувствует прилив сил (+%d STR)!", actor.Name, props.EffectValue)
	}
	return ""
}
