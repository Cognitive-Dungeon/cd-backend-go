package api

type SpellView struct {
	ID       uint32  `json:"id"`
	Name     string  `json:"name"`
	Cost     int     `json:"cost"`
	CostType string  `json:"costType"` // "MANA", "HP"
	Cooldown int     `json:"cooldown"` // Полный КД (для справки)
	Range    float64 `json:"range"`
}

// TileView — описание одной клетки карты.
type TileView struct {
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Symbol    string `json:"symbol"`
	Color     string `json:"color"`
	IsWall    bool   `json:"isWall"`
	IsVisible bool   `json:"isVisible"` // Для тумана войны
}

// EntityView — описание объекта для отрисовки.
type EntityView struct {
	ID   string `json:"id"`
	Type string `json:"type"` // "UNIT", "ITEM"
	Name string `json:"name"`

	Pos struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"pos"`

	Render struct {
		Symbol string `json:"symbol"`
		Color  string `json:"color"`
	} `json:"render"`

	Stats *StatsView `json:"stats,omitempty"`
}

// StatsView — характеристики (HP/Mana).
type StatsView struct {
	HP    int `json:"hp"`
	MaxHP int `json:"maxHp"`
}
