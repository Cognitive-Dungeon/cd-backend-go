package actions

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/pkg/api"
	"cognitive-server/pkg/logger"
	"fmt"

	"github.com/sirupsen/logrus"
)

// HandlePickup обрабатывает команду PICKUP - подбор предмета с земли
func HandlePickup(ctx handlers.Context, p api.ItemPayload) (handlers.Result, error) {
	actor := ctx.Actor

	log := logger.Log.WithFields(logrus.Fields{
		"component":  "pickup_handler",
		"actor_id":   actor.ID,
		"actor_name": actor.Name,
	})

	// Проверяем, есть ли у актора инвентарь
	if actor.Inventory == nil {
		log.Warn("Actor has no inventory component")
		return handlers.Result{
			Msg:     fmt.Sprintf("%s не может ничего подобрать.", actor.Name),
			MsgType: "ERROR",
		}, nil
	}

	// Находим предмет в мире
	item := ctx.Finder.GetEntity(p.ItemID)
	if item == nil {
		log.WithField("item_id", p.ItemID).Warn("Item not found")
		return handlers.Result{Msg: "Предмет не найден.", MsgType: "ERROR"}, nil
	}

	// Проверяем, что это предмет
	if item.Item == nil {
		log.WithField("item_id", item.ID).Warn("Entity is not an item")
		return handlers.Result{Msg: fmt.Sprintf("%s нельзя подобрать.", item.Name), MsgType: "ERROR"}, nil
	}

	// Проверяем, что предмет на том же уровне
	if item.Level != actor.Level {
		log.Debug("Item is on different level")
		return handlers.Result{Msg: "Предмет слишком далеко.", MsgType: "ERROR"}, nil
	}

	// Проверяем расстояние (должен быть рядом)
	distance := actor.Pos.DistanceSquaredTo(item.Pos)
	maxDistance := 2 // Рядом = в радиусе 1 клетки (1.5^2 = 2.25, округляем до 2)

	if distance > maxDistance {
		log.WithField("distance", distance).Debug("Item is too far away")
		return handlers.Result{Msg: "Предмет слишком далеко.", MsgType: "ERROR"}, nil
	}

	// Пытаемся добавить предмет в инвентарь
	if !actor.Inventory.AddItem(item) {
		log.Debug("Failed to add item to inventory (full or too heavy)")
		return handlers.Result{Msg: "Инвентарь полон или предмет слишком тяжёлый.", MsgType: "ERROR"}, nil
	}

	// Удаляем предмет из мира
	ctx.World.RemoveEntity(item)
	ctx.World.UnregisterEntity(item.ID)

	// Помечаем, что предмет больше не находится на уровне (уходит в Лимбо/Инвентарь)
	item.Level = -1

	// Списываем время
	actor.AI.NextActionTick += domain.TimeCostPickup

	// Логируем успех
	log.WithField("item_name", item.Name).Info("Item picked up successfully")
	return handlers.Result{
		Msg:     fmt.Sprintf("%s подбирает %s.", actor.Name, item.Name),
		MsgType: "INFO",
	}, nil
}
