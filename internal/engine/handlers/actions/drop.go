package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

// HandleDrop обрабатывает команду DROP - выброс предмета из инвентаря
func HandleDrop(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	actor := ctx.Actor

	log := logger.Log.WithFields(logrus.Fields{
		"component":  "drop_handler",
		"actor_id":   actor.ID,
		"actor_name": actor.Name,
	})

	// Проверяем, есть ли у актора инвентарь
	if actor.Inventory == nil {
		log.Warn("Actor has no inventory component")
		return handlers.Result{Msg: fmt.Sprintf("%s не может ничего выбросить.", actor.Name), MsgType: "ERROR"}, nil
	}

	// Если count не указан, выбрасываем весь стак
	count := p.Count
	if count <= 0 {
		count = 1
	}

	// Находим предмет в инвентаре
	item := actor.Inventory.FindItem(p.ItemID)
	if item == nil {
		log.WithField("item_id", p.ItemID).Warn("Item not found in inventory")
		return handlers.Result{Msg: "Предмет не найден в инвентаре.", MsgType: "ERROR"}, nil
	}

	// Обрабатываем стаки
	if item.Item != nil && item.Item.IsStackable && item.Item.StackSize > count {
		// Уменьшаем размер стака
		item.Item.StackSize -= count

		// Создаём новый предмет для выброса
		droppedItem := &domain.Entity{
			ID:     domain.GenerateID(),
			Type:   domain.EntityTypeItem,
			Name:   item.Name,
			Pos:    actor.Pos,
			Level:  actor.Level,
			Render: item.Render,
			Item: &domain.ItemComponent{
				Category:    item.Item.Category,
				IsStackable: item.Item.IsStackable,
				StackSize:   count,
				Weight:      item.Item.Weight,
				Value:       item.Item.Value,
			},
		}

		ctx.World.RegisterEntity(droppedItem)
		ctx.World.AddEntity(droppedItem)

		actor.AI.NextActionTick += domain.TimeCostDrop
		return handlers.Result{
			Msg:     fmt.Sprintf("%s выбрасывает %dx %s.", actor.Name, count, item.Name),
			MsgType: "INFO",
		}, nil
	}

	// Выбрасываем весь предмет
	actor.Inventory.RemoveItem(p.ItemID)
	item.Pos = actor.Pos
	item.Level = actor.Level

	ctx.World.RegisterEntity(item)
	ctx.World.AddEntity(item)

	actor.AI.NextActionTick += domain.TimeCostDrop

	log.WithField("item_name", item.Name).Info("Item dropped successfully")
	return handlers.Result{
		Msg:     fmt.Sprintf("%s выбрасывает %s.", actor.Name, item.Name),
		MsgType: "INFO",
	}, nil
}
