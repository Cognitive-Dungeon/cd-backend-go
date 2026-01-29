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

// IsEmpty проверяет, пустой ли тайл (MaterialID == 0).
func (t Tile) IsEmpty() bool {
	return t.Material == 0
}
