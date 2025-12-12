package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

// HandleUnequip обрабатывает команду UNEQUIP - снятие экипировки
func HandleUnequip(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	actor := ctx.Actor

	log := logger.Log.WithFields(logrus.Fields{
		"component":  "unequip_handler",
		"actor_id":   actor.ID,
		"actor_name": actor.Name,
	})

	if actor.Equipment == nil {
		log.Warn("Actor has no equipment component")
		return handlers.Result{Msg: fmt.Sprintf("%s не имеет экипировки.", actor.Name), MsgType: "ERROR"}, nil
	}

	var itemName string

	if actor.Equipment.Weapon != nil && actor.Equipment.Weapon.ID == p.ItemID {
		itemName = actor.Equipment.Weapon.Name
		actor.Equipment.Weapon = nil
	} else if actor.Equipment.Armor != nil && actor.Equipment.Armor.ID == p.ItemID {
		itemName = actor.Equipment.Armor.Name
		actor.Equipment.Armor = nil
	} else {
		log.WithField("item_id", p.ItemID).Warn("Item is not equipped")
		return handlers.Result{Msg: "Этот предмет не экипирован.", MsgType: "ERROR"}, nil
	}

	actor.AI.NextActionTick += domain.TimeCostUnequip

	log.WithField("item_name", itemName).Info("Item unequipped successfully")
	return handlers.Result{
		Msg:     fmt.Sprintf("%s снимает %s.", actor.Name, itemName),
		MsgType: "INFO",
	}, nil
}
