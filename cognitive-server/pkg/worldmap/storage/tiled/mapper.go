package tiled

import "cognitive-server/pkg/worldmap"

// TileDef описывает правила конвертации для одного GID (из вашей палитры).
type TileDef struct {
	MaterialID worldmap.MaterialID
	Flags      worldmap.TileFlag
	Variant    uint8
}

// Palette — словарь маппинга.
type Palette map[int]TileDef

const (
	// Битовые маски Tiled для трансформаций (Flipping)
	// См. https://doc.mapeditor.org/en/stable/reference/tmx-map-format/#tile-flipping
	maskHorizontal = 0x80000000
	maskVertical   = 0x40000000
	maskDiagonal   = 0x20000000
	maskHexagonal  = 0x10000000 // (optional)

	// Маска для получения чистого ID тайла (без флагов)
	maskGID = 0x0FFFFFFF
)

// mapGid преобразует GID из Tiled в серверный Tile.
func mapGid(rawGID uint32, palette Palette) (worldmap.Tile, bool) {
	if rawGID == 0 {
		// Default tile aka Void
		return worldmap.Tile{}, false
	}

	// Очищаем флаги поворота
	gid := int(rawGID & maskGID)

	def, ok := palette[gid]
	if !ok {
		return worldmap.Tile{}, false
	}

	// TODO: Если в будущем захотим поддерживать поворот стен,
	// можно извлекать флаги здесь:
	// flippedH := (rawGID & maskHorizontal) != 0
	// ... и записывать их в Variant или Flags тайла.

	return worldmap.Tile{
		Material: def.MaterialID,
		Flags:    def.Flags,
		Variant:  def.Variant,
	}, true
}
