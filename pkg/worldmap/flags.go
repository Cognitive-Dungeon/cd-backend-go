package worldmap

// TileFlag — битовая маска свойств тайла.
type TileFlag uint8

const (
	FlagNone   TileFlag = 0
	FlagSolid  TileFlag = 1 << 0 // Блокирует движение
	FlagOpaque TileFlag = 1 << 1 // Блокирует свет (FOV)
	FlagLiquid TileFlag = 1 << 2 // Жидкость (вода, лава)
	FlagGas    TileFlag = 1 << 3 // Газ (дым, пар)
)

// Has проверяет наличие флага.
func (f TileFlag) Has(flag TileFlag) bool {
	return f&flag != 0
}

// Add добавляет флаг (возвращает новую маску).
func (f TileFlag) Add(flag TileFlag) TileFlag {
	return f | flag
}

// Remove убирает флаг.
func (f TileFlag) Remove(flag TileFlag) TileFlag {
	return f &^ flag
}
