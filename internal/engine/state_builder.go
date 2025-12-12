package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

// publishUpdate рассылает актуальное состояние мира всем активным подписчикам КОНКРЕТНОГО инстанса.
func (s *GameService) publishUpdate(activeID string, instance *Instance) {
	// Пробегаем по сущностям ТОЛЬКО этого уровня
	for _, e := range instance.Entities {
		if s.Hub.HasSubscriber(e.ID) {
			state := s.BuildStateFor(e, activeID, instance)
			s.Hub.SendTo(e.ID, *state)
		}
	}

	// Очищаем логи инстанса после рассылки
	instance.Logs = []api.LogEntry{}
}

// BuildStateFor создает персональный "снимок" мира для конкретной сущности-наблюдателя.
func (s *GameService) BuildStateFor(observer *domain.Entity, activeID string, instance *Instance) *api.ServerResponse {
	// Используем мир из инстанса
	observerWorld := instance.World

	// 1. Расчет FOV (Поля зрения)
	var visibleIdxs map[int]bool
	isGod := false

	if observer.Vision != nil {
		visibleIdxs = systems.ComputeVisibleTiles(observerWorld, observer.Pos, observer.Vision)
		if visibleIdxs == nil {
			isGod = true
		}
	}

	// Обновляем память (туман войны)
	if observer.Memory != nil && !isGod && visibleIdxs != nil {
		currentLevelMemory := observer.Memory.ExploredPerLevel[observer.Level]
		if currentLevelMemory == nil {
			currentLevelMemory = make(map[int]bool)
			observer.Memory.ExploredPerLevel[observer.Level] = currentLevelMemory
		}
		for idx := range visibleIdxs {
			currentLevelMemory[idx] = true
		}
	}

	// 2. Формирование карты (Map DTO)
	var mapDTO []api.TileView
	for y := 0; y < observerWorld.Height; y++ {
		for x := 0; x < observerWorld.Width; x++ {
			idx := observerWorld.GetIndex(x, y)

			isExplored := isGod
			if !isGod && observer.Memory != nil {
				if levelMemory, ok := observer.Memory.ExploredPerLevel[observer.Level]; ok {
					isExplored = levelMemory[idx]
				}
			}

			if isExplored {
				tile := observerWorld.Map[y][x]
				isVisible := isGod || visibleIdxs[idx]

				tView := api.TileView{
					X: x, Y: y, IsWall: tile.IsWall,
					IsVisible:  isVisible,
					IsExplored: true,
					Symbol:     ".", Color: "#333333",
				}
				if tile.IsWall {
					tView.Symbol = "#"
					tView.Color = "#666666"
				}
				mapDTO = append(mapDTO, tView)
			}
		}
	}

	// 3. Формирование списка сущностей (Entities DTO)
	// Берем сущности из INSTANCE, а не из Service
	var viewEntities []api.EntityView

	for _, e := range instance.Entities {
		// Себя видим всегда
		if e.ID == observer.ID {
			viewEntities = append(viewEntities, s.toEntityView(e, observer))
			continue
		}

		// Проверка реестра (на случай рассинхрона)
		if observerWorld.GetEntity(e.ID) == nil {
			continue
		}

		// Остальных - если они в зоне видимости
		idx := observerWorld.GetIndex(e.Pos.X, e.Pos.Y)
		if isGod || visibleIdxs[idx] {
			viewEntities = append(viewEntities, s.toEntityView(e, observer))
		}
	}

	// Копия логов
	logsCopy := make([]api.LogEntry, len(instance.Logs))
	copy(logsCopy, instance.Logs)

	return &api.ServerResponse{
		Type:           "UPDATE",
		Tick:           0, // Tick теперь локальный для инстанса, можно передать instance.CurrentTick, если нужно
		MyEntityID:     observer.ID,
		ActiveEntityID: activeID,
		Grid:           &api.GridMeta{Width: observerWorld.Width, Height: observerWorld.Height},
		Map:            mapDTO,
		Entities:       viewEntities,
		Logs:           logsCopy,
	}
}

// toEntityView конвертирует доменную сущность в DTO для отправки клиенту.
func (s *GameService) toEntityView(target *domain.Entity, observer *domain.Entity) api.EntityView {
	view := api.EntityView{
		ID:   target.ID,
		Type: target.Type,
		Name: target.Name,
	}
	view.Pos.X = target.Pos.X
	view.Pos.Y = target.Pos.Y

	if target.Render != nil {
		view.Render.Symbol = target.Render.Symbol
		view.Render.Color = target.Render.Color
	} else {
		view.Render.Symbol = "?"
		view.Render.Color = "#fff"
	}

	// Логика видимости статов
	isMe := target.ID == observer.ID
	isDead := target.Stats != nil && target.Stats.IsDead

	if target.Stats != nil {
		if isMe {
			// Владелец видит всё
			view.Stats = &api.StatsView{
				HP: target.Stats.HP, MaxHP: target.Stats.MaxHP,
				Stamina: target.Stats.Stamina, MaxStamina: target.Stats.MaxStamina,
				Gold: target.Stats.Gold, Strength: target.Stats.Strength,
				IsDead: target.Stats.IsDead,
			}
		} else {
			// Чужаки видят минимум (можно добавить Perception Check здесь)
			view.Stats = &api.StatsView{
				HP: target.Stats.HP, MaxHP: target.Stats.MaxHP,
				IsDead: target.Stats.IsDead,
			}
		}
	}

	if isDead {
		view.Stats.IsDead = true
	}

	// Инвентарь и экипировка (только владелец и контейнеры)
	if isMe || target.Type == domain.EntityTypeItem {
		// Инвентарь
		if target.Inventory != nil {
			invView := &api.InventoryView{
				Items:         make([]api.ItemView, 0, len(target.Inventory.Items)),
				MaxSlots:      target.Inventory.MaxSlots,
				CurrentWeight: target.Inventory.CurrentWeight,
				MaxWeight:     target.Inventory.MaxWeight,
			}

			for _, item := range target.Inventory.Items {
				if item != nil && item.Item != nil {
					itemView := api.ItemView{
						ID:          item.ID,
						Name:        item.Name,
						Category:    item.Item.Category,
						IsStackable: item.Item.IsStackable,
						StackSize:   item.Item.StackSize,
						Damage:      item.Item.Damage,
						Defense:     item.Item.Defense,
						Weight:      item.Item.Weight,
						Value:       item.Item.Value,
						IsSentient:  item.Item.IsSentient,
					}

					if item.Render != nil {
						itemView.Symbol = item.Render.Symbol
						itemView.Color = item.Render.Color
					}

					invView.Items = append(invView.Items, itemView)
				}
			}

			view.Inventory = invView
		}

		// Экипировка (только для владельца)
		if isMe && target.Equipment != nil {
			eqView := &api.EquipmentView{}

			if target.Equipment.Weapon != nil && target.Equipment.Weapon.Item != nil {
				w := target.Equipment.Weapon
				weaponView := api.ItemView{
					ID:       w.ID,
					Name:     w.Name,
					Category: w.Item.Category,
					Damage:   w.Item.Damage,
					Weight:   w.Item.Weight,
					Value:    w.Item.Value,
				}
				if w.Render != nil {
					weaponView.Symbol = w.Render.Symbol
					weaponView.Color = w.Render.Color
				}
				eqView.Weapon = &weaponView
			}

			if target.Equipment.Armor != nil && target.Equipment.Armor.Item != nil {
				a := target.Equipment.Armor
				armorView := api.ItemView{
					ID:       a.ID,
					Name:     a.Name,
					Category: a.Item.Category,
					Defense:  a.Item.Defense,
					Weight:   a.Item.Weight,
					Value:    a.Item.Value,
				}
				if a.Render != nil {
					armorView.Symbol = a.Render.Symbol
					armorView.Color = a.Render.Color
				}
				eqView.Armor = &armorView
			}

			view.Equipment = eqView
		}
	}

	return view
}
