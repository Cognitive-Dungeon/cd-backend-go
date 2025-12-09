package engine

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/systems"
	"cognitive-server/pkg/api"
)

// publishUpdate рассылает актуальное состояние мира всем активным подписчикам.
func (s *GameService) publishUpdate(activeID string) {
	// Пробегаем по всем сущностям, и если у них есть "душа" (подключенный клиент), шлем апдейт
	for _, e := range s.Entities {
		if s.Hub.HasSubscriber(e.ID) {
			state := s.BuildStateFor(e, activeID)
			s.Hub.SendTo(e.ID, *state)
		}
	}

	// Очищаем логи ПОСЛЕ рассылки (так как они рассылаются всем одинаковые в текущей итерации)
	// Примечание: В production лучше хранить лог буфер или рассылать события сразу.
	s.Logs = []api.LogEntry{}
}

// BuildStateFor создает персональный "снимок" мира для конкретной сущности-наблюдателя.
func (s *GameService) BuildStateFor(observer *domain.Entity, activeID string) *api.ServerResponse {
	observerWorld, ok := s.Worlds[observer.Level]
	if !ok {
		return &api.ServerResponse{Type: "ERROR", Logs: []api.LogEntry{{Text: "You are in the void."}}}
	}

	// 1. Расчет FOV (Поля зрения)
	var visibleIdxs map[int]bool
	isGod := false

	if observer.Vision != nil {
		visibleIdxs = systems.ComputeVisibleTiles(observerWorld, observer.Pos, observer.Vision)
		if visibleIdxs == nil { // nil возвращается для Omniscient (всевидящих)
			isGod = true
		}
	}

	// Обновляем память (туман войны)
	if observer.Memory != nil && !isGod && visibleIdxs != nil {
		// Работаем с памятью текущего уровня ---
		currentLevelMemory := observer.Memory.ExploredPerLevel[observer.Level]
		// Если для этого уровня памяти еще нет, создаем ее
		if currentLevelMemory == nil {
			currentLevelMemory = make(map[int]bool)
			observer.Memory.ExploredPerLevel[observer.Level] = currentLevelMemory
		}

		// Записываем все видимые сейчас тайлы в память ТЕКУЩЕГО уровня
		for idx := range visibleIdxs {
			currentLevelMemory[idx] = true
		}
	}

	// 2. Формирование карты (Map DTO)
	var mapDTO []api.TileView
	// TODO: Оптимизация: можно отправлять только изменения, но пока шлем всю видимую карту
	for y := 0; y < observerWorld.Height; y++ {
		for x := 0; x < observerWorld.Width; x++ {
			idx := observerWorld.GetIndex(x, y)

			// Проверяем, знает ли наблюдатель об этой клетке
			isExplored := isGod
			if !isGod && observer.Memory != nil {
				if levelMemory, ok := observer.Memory.ExploredPerLevel[observer.Level]; ok {
					isExplored = levelMemory[idx]
				}
			}

			// Если клетка исследована, добавляем её в ответ
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
	var viewEntities []api.EntityView

	for _, e := range s.Entities {
		// --- ВАЖНО: Показываем только сущностей на том же уровне ---
		if e.Level != observer.Level {
			continue
		}

		// Себя видим всегда
		if e.ID == observer.ID {
			viewEntities = append(viewEntities, s.toEntityView(e, observer))
			continue
		}

		// Остальных - если они в зоне видимости
		idx := observerWorld.GetIndex(e.Pos.X, e.Pos.Y)
		if isGod || visibleIdxs[idx] {
			viewEntities = append(viewEntities, s.toEntityView(e, observer))
		}
	}

	// Копия логов, чтобы не было гонки данных
	logsCopy := make([]api.LogEntry, len(s.Logs))
	copy(logsCopy, s.Logs)

	return &api.ServerResponse{
		Type:           "UPDATE",
		Tick:           s.GlobalTick,
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

	return view
}
