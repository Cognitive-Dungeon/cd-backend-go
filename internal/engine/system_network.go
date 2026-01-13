package engine

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
)

// GetWorldSnapshot создает DTO состояния мира для конкретного игрока.
// В будущем тут будет расчет FOV (тумана войны). Пока шлем всё.
func (e *Engine) GetWorldSnapshot(playerGuid ObjectGuid) *api.ServerResponse {
	resp := &api.ServerResponse{
		Type: "UPDATE",
		Tick: 0, // TODO: e.TickCount
		Grid: &api.GridMeta{Width: int(e.Instance.Grid.Width), Height: int(e.Instance.Grid.Height)},
	}

	// 1. Рисуем карту (Тайлы)
	// В реальном проекте карту шлем 1 раз или чанками. Для MVP шлем каждый кадр.
	for y := int32(0); y < int32(e.Instance.Grid.Height); y++ {
		for x := int32(0); x < int32(e.Instance.Grid.Width); x++ {
			tileType := enums.TileFloor
			if !e.Instance.Grid.IsWalkable(TilePos{X: types.TileCoord(x), Y: types.TileCoord(y)}) {
				tileType = enums.TileWall
			}

			view := api.TileView{
				X: int(x), Y: int(y),
				IsVisible: true,
			}

			if tileType == enums.TileWall {
				view.Symbol = "#"
				view.Color = "#555"
				view.IsWall = true
			} else {
				view.Symbol = "."
				view.Color = "#222"
			}
			resp.Map = append(resp.Map, view)
		}
	}

	// 2. Рисуем Сущности
	// Пробегаем по всем чанкам
	for chunkIdx, chunk := range e.Instance.Guids {
		for slotIdx, guid := range chunk {
			if guid == 0 { // NilObjectGuid
				continue
			}

			// Получаем компоненты напрямую по индексам
			pos := e.Instance.Positions[chunkIdx][slotIdx]
			render := e.Instance.Renders[chunkIdx][slotIdx]
			stats := e.Instance.Stats[chunkIdx][slotIdx]
			name := e.Instance.Names[chunkIdx][slotIdx]

			if pos == nil || render == nil {
				continue
			}

			entView := api.EntityView{
				ID:   guid.String(),
				Name: "Unknown",
				Type: "NONE",
			}

			if name != nil {
				entView.Name = name.Name
			}

			entView.Pos.X = int(pos.X)
			entView.Pos.Y = int(pos.Y)
			entView.Render.Symbol = string([]byte{render.Glyph.Char()})
			entView.Render.Color = render.Glyph.HexColor()

			if stats != nil {
				entView.Stats = &api.StatsView{
					HP:    int(stats.Health),
					MaxHP: int(stats.MaxHealth),
				}
			}

			if guid == playerGuid {
				resp.MyEntityID = guid.String()
				resp.ActiveEntityID = guid.String() // Разрешаем ввод
			}

			resp.Entities = append(resp.Entities, entView)
		}
	}

	return resp
}
