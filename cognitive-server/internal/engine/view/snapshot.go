package view

import (
	"cognitive-server/internal/api"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/engine"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/components"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/worldmap"
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
	wm := inst.WorldMap
	matReg := b.Engine.MaterialRegistry

	// Область видимости для снапшота
	// TODO: Убрать после реализации FOV
	// TEMP
	viewW, viewH := 40, 40

	resp := &api.ServerResponse{
		Type: "UPDATE",
		Tick: 0,
		Grid: &api.GridMeta{Width: viewW, Height: viewH},
	}

	// 1. Карта (без изменений)
	for y := 0; y < viewH; y++ {
		for x := 0; x < viewW; x++ {
			pos := geo.Pos(x, y, 0)
			tile := wm.GetTile(pos)
			vis := matReg.GetVisual(tile.Material)

			view := api.TileView{
				X:         x,
				Y:         y,
				Symbol:    vis.Glyph.CharUTF8(),
				Color:     vis.Glyph.HexColor(),
				IsWall:    tile.Flags.Has(worldmap.FlagSolid),
				IsVisible: true,
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
