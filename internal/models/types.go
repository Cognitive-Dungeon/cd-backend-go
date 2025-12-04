package models

import "encoding/json"

// --- Константы ---
const (
	EntityTypePlayer = "PLAYER"
	EntityTypeNPC    = "NPC"
	EntityTypeEnemy  = "ENEMY"
	EntityTypeItem   = "ITEM"
	EntityTypeExit   = "EXIT"
)

// --- Базовые структуры ---

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Stats struct {
	HP         int `json:"hp"`
	MaxHP      int `json:"maxHp"`
	Stamina    int `json:"stamina"`
	MaxStamina int `json:"maxStamina"`
	Gold       int `json:"gold"`
	Strength   int `json:"strength"`
}

type Entity struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"` // A, B, @
	Type      string   `json:"type"`  // PLAYER, NPC
	Symbol    string   `json:"symbol"`
	Color     string   `json:"color"`
	Pos       Position `json:"pos"`
	Stats     Stats    `json:"stats"`
	Name      string   `json:"name"`
	IsHostile bool     `json:"isHostile"`
	IsDead    bool     `json:"isDead"`

	// AI & Time System
	NextActionTick int    `json:"nextActionTick"`
	Personality    string `json:"personality,omitempty"`
	AIState        string `json:"aiState,omitempty"`
}

type Tile struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	IsWall     bool   `json:"isWall"`
	Env        string `json:"env"` // stone, grass, water
	IsVisible  bool   `json:"isVisible"`
	IsExplored bool   `json:"isExplored"`
}

type GameWorld struct {
	Map        [][]Tile `json:"map"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Level      int      `json:"level"`
	GlobalTick int      `json:"globalTick"`
}

// --- Сетевые сообщения ---

// LogEntry - одна строка в чате
type LogEntry struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Type      string `json:"type"` // INFO, COMBAT, SPEECH
	Timestamp int64  `json:"timestamp"`
}

// ServerResponse - то, что летит на фронтенд
type ServerResponse struct {
	Type     string     `json:"type"` // INIT, UPDATE
	World    *GameWorld `json:"world,omitempty"`
	Player   *Entity    `json:"player,omitempty"`
	Entities []Entity   `json:"entities,omitempty"`
	Logs     []LogEntry `json:"logs,omitempty"`
}

// ClientCommand - то, что прилетает с фронтенда
type ClientCommand struct {
	Action  string          `json:"action"` // MOVE, ATTACK
	Payload json.RawMessage `json:"payload"`
}

// MovePayload - Payload для команды MOVE
type MovePayload struct {
	Dx int `json:"dx"`
	Dy int `json:"dy"`
}
