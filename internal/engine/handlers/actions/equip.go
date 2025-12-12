package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

// HandleEquip обрабатывает команду EQUIP - экипировка оружия/брони
func HandleEquip(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	actor := ctx.Actor

	log := logger.Log.WithFields(logrus.Fields{
		"component":  "equip_handler",
		"actor_id":   actor.ID,
		"actor_name": actor.Name,
	})

	if actor.Inventory == nil {
		log.Warn("Actor has no inventory component")
		return handlers.Result{Msg: fmt.Sprintf("%s не может экипировать предметы.", actor.Name), MsgType: "ERROR"}, nil
	}

	if actor.Equipment == nil {
		actor.Equipment = &domain.EquipmentComponent{}
	}

	item := actor.Inventory.FindItem(p.ItemID)
	if item == nil {
		log.WithField("item_id", p.ItemID).Warn("Item not found in inventory")
		return handlers.Result{Msg: "Предмет не найден в инвентаре.", MsgType: "ERROR"}, nil
	}

	if item.Item == nil {
		log.Warn("Item has no ItemComponent")
		return handlers.Result{Msg: "Этот предмет нельзя экипировать.", MsgType: "ERROR"}, nil
	}

	var oldItem *domain.Entity
	equipped := false

	switch item.Item.Category {
	case domain.ItemCategoryWeapon:
		if actor.Equipment.Weapon != nil {
			oldItem = actor.Equipment.Weapon
		}
		actor.Equipment.Weapon = item
		equipped = true

	case domain.ItemCategoryArmor:
		if actor.Equipment.Armor != nil {
			oldItem = actor.Equipment.Armor
		}
		actor.Equipment.Armor = item
		equipped = true

	default:
		log.WithField("category", item.Item.Category).Warn("Item cannot be equipped")
		return handlers.Result{Msg: fmt.Sprintf("%s нельзя экипировать.", item.Name), MsgType: "ERROR"}, nil
	}

	if !equipped {
		log.Error("Failed to equip item")
		return handlers.Result{Msg: "Не удалось экипировать предмет.", MsgType: "ERROR"}, nil
	}

	actor.AI.NextActionTick += domain.TimeCostEquip

	var message string
	if oldItem != nil {
		message = fmt.Sprintf("%s снимает %s и экипирует %s.", actor.Name, oldItem.Name, item.Name)
	} else {
		message = fmt.Sprintf("%s экипирует %s.", actor.Name, item.Name)
	}

	log.WithField("item_name", item.Name).Info("Item equipped successfully")
	return handlers.Result{Msg: message, MsgType: "INFO"}, nil
}
