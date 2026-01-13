package view

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine"
)

// NewSnapshotBuilder создает хелпер для генерации ответов
type SnapshotBuilder struct {
	Engine *engine.Engine
}

func New(eng *engine.Engine) *SnapshotBuilder {
	return &SnapshotBuilder{Engine: eng}
}

// BuildSnapshot создает DTO состояния мира
func (b *SnapshotBuilder) BuildSnapshot(playerGuid engine.ObjectGuid) *api.ServerResponse {
	inst := b.Engine.Instance

	resp := &api.ServerResponse{
		Type: "UPDATE",
		Tick: 0, // TODO: брать из Engine
		Grid: &api.GridMeta{Width: int(inst.Grid.Width), Height: int(inst.Grid.Height)},
	}

	// 1. Карта
	for y := int32(0); y < int32(inst.Grid.Height); y++ {
		for x := int32(0); x < int32(inst.Grid.Width); x++ {
			tileType := enums.TileFloor
			if !inst.Grid.IsWalkable(engine.TilePos{X: types.TileCoord(x), Y: types.TileCoord(y)}) {
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

	// 2. Сущности
	for chunkIdx, chunk := range inst.Guids {
		for slotIdx, guid := range chunk {
			if guid == 0 {
				continue
			}

			pos := inst.Positions[chunkIdx][slotIdx]
			render := inst.Renders[chunkIdx][slotIdx]
			stats := inst.Stats[chunkIdx][slotIdx]
			name := inst.Names[chunkIdx][slotIdx]

			if pos == nil || render == nil {
				continue
			}

			entView := api.EntityView{
				ID:   guid.String(),
				Name: "Unknown",
				Type: "UNIT",
			}

			if name != nil {
				entView.Name = name.Name
			}

			entView.Pos.X = int(pos.TilePos.X)
			entView.Pos.Y = int(pos.TilePos.Y)
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
				resp.ActiveEntityID = guid.String()
			}

			resp.Entities = append(resp.Entities, entView)
		}
	}

	return resp
}
