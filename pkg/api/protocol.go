package api

import (
	"encoding/json"
)

// --- СЕРВЕР -> КЛИЕНТ ---

// ServerResponse это корневой объект, который сервер отправляет клиенту.
// Он представляет собой полный "снимок" мира, видимого для конкретного клиента.
// Отправляется каждый раз, когда наступает ход сущности, которой управляет клиент.
type ServerResponse struct {
	// Type тип сообщения. На данный момент всегда "UPDATE".
	Type string `json:"type"`

	// Tick текущее глобальное время в игре. Увеличивается с каждым ходом.
	Tick int `json:"tick"`

	// ActiveEntityID ID сущности, чей ход сейчас.
	// КЛИЕНТ ДОЛЖЕН СРАВНИВАТЬ ЭТО ПОЛЕ СО СВОИМ ID. Если они совпадают,
	// значит, можно принимать ввод от игрока.
	ActiveEntityID string `json:"activeEntityId,omitempty"`

	// MyEntityID ID сущности, которой управляет данный клиент.
	MyEntityID string `json:"myEntityId,omitempty"`

	// Grid метаданные о размере всей карты.
	Grid *GridMeta `json:"grid,omitempty"`

	// Map срез всех видимых и/или исследованных тайлов.
	Map []TileView `json:"map,omitempty"`

	// Entities срез всех видимых сущностей.
	Entities []EntityView `json:"entities,omitempty"`

	// Logs срез новых сообщений, сгенерированных с прошлого хода.
	Logs []LogEntry `json:"logs,omitempty"`
}

// GridMeta содержит общие размеры карты, чтобы клиент знал,
// какую сетку для рендеринга нужно подготовить.
type GridMeta struct {
	Width  int `json:"w"`
	Height int `json:"h"`
}

// TileView это DTO (Data Transfer Object) для одного тайла карты.
// Содержит всю необходимую информацию для его рендеринга.
type TileView struct {
	X int `json:"x"`
	Y int `json:"y"`

	// Symbol и Color - визуальное представление тайла (e.g. "#" для стены).
	Symbol string `json:"symbol"`
	Color  string `json:"color"`

	// IsWall true, если тайл является непроходимым препятствием.
	IsWall bool `json:"isWall"`

	// IsVisible true, если тайл находится в текущем поле зрения. Рендерится ярко.
	IsVisible bool `json:"isVisible"`

	// IsExplored true, если тайл когда-либо был увиден. Используется для "тумана войны".
	// Если IsVisible=false, а IsExplored=true, рендерится тускло.
	IsExplored bool `json:"isExplored"`
}

// EntityView это DTO для игровой сущности.
type EntityView struct {
	ID   string `json:"id"`
	Type string `json:"type"` // PLAYER, ENEMY, NPC, ITEM
	Name string `json:"name"`

	Pos struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"pos"`

	Render struct {
		Symbol string `json:"symbol"`
		Color  string `json:"color"`
	} `json:"render"`

	// Stats характеристики сущности. Поле может отсутствовать (omitempty),
	// если клиент не имеет права видеть статы этой сущности.
	Stats *StatsView `json:"stats,omitempty"`

	// Inventory инвентарь сущности (для игрока и контейнеров)
	Inventory *InventoryView `json:"inventory,omitempty"`

	// Equipment экипированные предметы
	Equipment *EquipmentView `json:"equipment,omitempty"`
}

// StatsView это DTO для характеристик сущности.
// Некоторые поля могут отсутствовать, если сервер скрывает их от клиента.
type StatsView struct {
	HP         int  `json:"hp"`
	MaxHP      int  `json:"maxHp"`
	Stamina    int  `json:"stamina,omitempty"`
	MaxStamina int  `json:"maxStamina,omitempty"`
	Gold       int  `json:"gold,omitempty"`
	Strength   int  `json:"strength,omitempty"`
	IsDead     bool `json:"isDead"`
}

// LogEntry представляет одну запись в игровом логе (чате).
type LogEntry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Type      string `json:"type"`      // INFO, COMBAT, SPEECH, ERROR
	Timestamp int64  `json:"timestamp"` // Unix milliseconds
}

// ItemView представляет предмет для клиента
type ItemView struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Color       string `json:"color"`
	Category    string `json:"category"`
	IsStackable bool   `json:"isStackable,omitempty"`
	StackSize   int    `json:"stackSize,omitempty"`
	Damage      int    `json:"damage,omitempty"`
	Defense     int    `json:"defense,omitempty"`
	Weight      int    `json:"weight,omitempty"`
	Value       int    `json:"value,omitempty"`
	IsSentient  bool   `json:"isSentient,omitempty"`
}

// InventoryView представляет инвентарь для клиента
type InventoryView struct {
	Items         []ItemView `json:"items"`
	MaxSlots      int        `json:"maxSlots"`
	CurrentWeight int        `json:"currentWeight"`
	MaxWeight     int        `json:"maxWeight,omitempty"`
}

// EquipmentView представляет экипированные предметы
type EquipmentView struct {
	Weapon *ItemView `json:"weapon,omitempty"`
	Armor  *ItemView `json:"armor,omitempty"`
}

// --- КЛИЕНТ -> СЕРВЕР ---

// ClientCommand это корневой объект для всех сообщений от клиента к серверу.
type ClientCommand struct {
	// Token ID сущности, от имени которой выполняется действие.
	// Обязателен только для первого сообщения "LOGIN".
	Token string `json:"token,omitempty"`

	// Action название действия, которое нужно выполнить.
	Action string `json:"action"`

	// Payload JSON-объект с данными для действия. Его структура зависит от Action.
	Payload json.RawMessage `json:"payload"`
}

// --- Payloads ---

// DirectionPayload используется для действий, связанных с направлением (e.g. MOVE).
type DirectionPayload struct {
	Dx int `json:"dx"` // Смещение по X (-1, 0, 1)
	Dy int `json:"dy"` // Смещение по Y (-1, 0, 1)
}

// EntityPayload используется для действий, нацеленных на другую сущность (e.g. ATTACK, TALK).
type EntityPayload struct {
	TargetID string `json:"targetId"`
}

// PositionPayload используется для действий, нацеленных на точку на карте (e.g. TELEPORT).
type PositionPayload struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// ItemPayload используется для действий с предметами (PICKUP, DROP, USE, EQUIP, UNEQUIP).
type ItemPayload struct {
	ItemID string `json:"itemId"`
	Count  int    `json:"count,omitempty"` // Для DROP - количество предметов в стаке
}
