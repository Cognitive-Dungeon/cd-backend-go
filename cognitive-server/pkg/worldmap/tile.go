package worldmap

// MaterialID — идентификатор типа материала.
type MaterialID uint16

// Tile — базовая единица карты.
// Размер: 2 (ID) + 1 (Flags) + 1 (Variant) = 4 байта.
type Tile struct {
	Material MaterialID
	Flags    TileFlag
	Variant  uint8 // Для визуального разнообразия (например, 4 варианта текстуры травы)
}

// Pack упаковывает тайл в uint32 для быстрого сравнения и кеширования.
// Layout: [ Variant (8) | Flags (8) | Material (16) ]
func (t Tile) Pack() uint32 {
	return (uint32(t.Variant) << 24) |
		(uint32(t.Flags) << 16) |
		uint32(t.Material)
}

func UnpackTile(packed uint32) Tile {
	return Tile{
		Material: MaterialID(packed & 0xFFFF),
		Flags:    TileFlag((packed >> 16) & 0xFF),
		Variant:  uint8((packed >> 24) & 0xFF),
	}
}

// IsEmpty проверяет, пустой ли тайл (MaterialID == 0).
func (t Tile) IsEmpty() bool {
	return t.Material == 0
}
