package view

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/engine/data"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/grid"
	"strconv"
)

// NewSnapshotBuilder создает хелпер для генерации ответов
type SnapshotBuilder struct {
	Engine *engine.Engine
}

func New(eng *engine.Engine) *SnapshotBuilder {
	return &SnapshotBuilder{Engine: eng}
}

// BuildSnapshot создает DTO состояния мира
func (b *SnapshotBuilder) BuildSnapshot(playerGuid ecs2.ObjectGuid) *api.ServerResponse {
	inst := b.Engine.Instance
	w := inst.World

	resp := &api.ServerResponse{
		Type: "UPDATE",
		Tick: 0,
		Grid: &api.GridMeta{Width: int(inst.Grid.Width), Height: int(inst.Grid.Height)},
	}

	// 1. Карта (без изменений)
	for y := int32(0); y < int32(inst.Grid.Height); y++ {
		for x := int32(0); x < int32(inst.Grid.Width); x++ {
			tileType := enums.TileFloor
			if !inst.Grid.IsWalkable(data.TilePos{X: grid.TileCoord(x), Y: grid.TileCoord(y)}) {
				tileType = enums.TileWall
			}
			view := api.TileView{X: int(x), Y: int(y), IsVisible: true}
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
	// Итерируемся по всем, у кого есть Position и Render.
	// Это аналог "Select * from Entities where Position and Render"
	for id, join := range ecs.View2[components.PositionComponent, components.RenderComponent](w, components.CID_Position, components.CID_Render) {
		pos := join.First
		render := join.Second

		guid := types.ObjectGuid(id) // Конвертация обратно

		entView := api.EntityView{
			ID:   strconv.FormatUint(uint64(guid), 10),
			Name: "Unknown",
			Type: "UNIT",
		}

		entView.Pos.X = int(pos.X)
		entView.Pos.Y = int(pos.Y)
		entView.Render.Symbol = string([]byte{render.Glyph.Char()})
		entView.Render.Color = render.Glyph.HexColor()

		// Опционально: Имя
		if name := ecs.GetStorage[components.NameComponent](w, components.CID_Name).Get(id); name != nil {
			entView.Name = name.Name
		}

		// Опционально: Статы
		if stats := ecs.GetStorage[components.StatsComponent](w, components.CID_Stats).Get(id); stats != nil {
			entView.Stats = &api.StatsView{
				HP:    int(stats.Health),
				MaxHP: int(stats.MaxHealth),
			}
		}

		// Логика для текущего игрока
		if guid == playerGuid {
			resp.MyEntityID = entView.ID
			resp.ActiveEntityID = guid.String()

			// Spellbook
			if sb := ecs.GetStorage[components.SpellbookComponent](w, components.CID_Spellbook).Get(id); sb != nil {
				for _, spellID := range sb.KnownSpells {
					info, found := b.Engine.SpellRegistry.Get(types.SpellID(spellID))
					if !found {
						continue
					}

					sView := api.SpellView{
						ID: uint32(info.ID), Name: info.Name, Cost: int(info.CostValue),
						Range: info.Range, Cooldown: int(info.Cooldown),
					}
					if info.CostType == types.SpellResourceMana {
						sView.CostType = "MANA"
					}
					resp.Spells = append(resp.Spells, sView)
				}
			}
		}

		resp.Entities = append(resp.Entities, entView)
	}

	return resp
}
