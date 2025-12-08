package domain

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Tile struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	IsWall     bool   `json:"isWall"`
	Env        string `json:"env"` // floor, stone, grass
	IsVisible  bool   `json:"isVisible"`
	IsExplored bool   `json:"isExplored"`

	// В будущем сюда добавятся ссылки на предметы на полу
}

type GameWorld struct {
	Map        [][]Tile `json:"map"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Level      int      `json:"level"`
	GlobalTick int      `json:"globalTick"`

	// SpatialHash: Индекс позиции -> Список сущностей
	// Ключ: Y * Width + X
	// json:"-" означает, что мы НЕ отправляем этот индекс клиенту (экономия трафика)
	SpatialHash    map[int][]*Entity  `json:"-"`
	EntityRegistry map[string]*Entity `json:"-"`
}
