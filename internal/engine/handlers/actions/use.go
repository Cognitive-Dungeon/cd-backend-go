package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

// HandleUse обрабатывает команду USE - использование предмета (зелья, еда)
func HandleUse(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	actor := ctx.Actor

	log := logger.Log.WithFields(logrus.Fields{
		"component":  "use_handler",
		"actor_id":   actor.ID,
		"actor_name": actor.Name,
	})

	if actor.Inventory == nil {
		log.Warn("Actor has no inventory component")
		return handlers.Result{Msg: fmt.Sprintf("%s не может ничего использовать.", actor.Name), MsgType: "ERROR"}, nil
	}

	item := actor.Inventory.FindItem(p.ItemID)
	if item == nil {
		log.WithField("item_id", p.ItemID).Warn("Item not found in inventory")
		return handlers.Result{Msg: "Предмет не найден в инвентаре.", MsgType: "ERROR"}, nil
	}

	if item.Item == nil || item.Item.EffectType == "" {
		log.WithField("item_name", item.Name).Warn("Item has no effect")
		return handlers.Result{Msg: fmt.Sprintf("%s нельзя использовать.", item.Name), MsgType: "ERROR"}, nil
	}

	effectApplied := false
	effectMessage := ""

	switch item.Item.EffectType {
	case "heal":
		if actor.Stats != nil {
			oldHP := actor.Stats.HP
			actor.Stats.HP += item.Item.EffectValue
			if actor.Stats.HP > actor.Stats.MaxHP {
				actor.Stats.HP = actor.Stats.MaxHP
			}
			healed := actor.Stats.HP - oldHP
			effectMessage = fmt.Sprintf("%s восстанавливает %d HP.", actor.Name, healed)
			effectApplied = true
		}

	case "restore_stamina":
		if actor.Stats != nil {
			oldStamina := actor.Stats.Stamina
			actor.Stats.Stamina += item.Item.EffectValue
			if actor.Stats.Stamina > actor.Stats.MaxStamina {
				actor.Stats.Stamina = actor.Stats.MaxStamina
			}
			restored := actor.Stats.Stamina - oldStamina
			effectMessage = fmt.Sprintf("%s восстанавливает %d выносливости.", actor.Name, restored)
			effectApplied = true
		}

	case "buff_strength":
		if actor.Stats != nil {
			actor.Stats.Strength += item.Item.EffectValue
			effectMessage = fmt.Sprintf("%s становится сильнее (+%d)!", actor.Name, item.Item.EffectValue)
			effectApplied = true
		}

	default:
		log.WithField("effect_type", item.Item.EffectType).Warn("Unknown effect type")
		return handlers.Result{Msg: fmt.Sprintf("Неизвестный эффект: %s", item.Item.EffectType), MsgType: "ERROR"}, nil
	}

	if !effectApplied {
		log.Error("Effect was not applied")
		return handlers.Result{Msg: "Не удалось применить эффект.", MsgType: "ERROR"}, nil
	}

	if item.Item.IsConsumable {
		if item.Item.IsStackable && item.Item.StackSize > 1 {
			item.Item.StackSize--
		} else {
			actor.Inventory.RemoveItem(p.ItemID)
		}
	}

	actor.AI.NextActionTick += domain.TimeCostUse

	log.WithFields(logrus.Fields{
		"item_name":   item.Name,
		"effect_type": item.Item.EffectType,
	}).Info("Item used successfully")

	return handlers.Result{Msg: effectMessage, MsgType: "INFO"}, nil
}
