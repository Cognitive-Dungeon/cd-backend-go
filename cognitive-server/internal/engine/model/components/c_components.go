package components

import (
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/grid"
	"cognitive-server/pkg/types/glyph"
)

type ComponentID = int

var (
	// State (Persistent)
	CID_Position, CID_Render, CID_Stats, CID_Name, CID_Spellbook, CID_Controller ComponentID

	// Input (Request)
	CID_CmdMove, CID_CmdCast ComponentID

	// Logic (Intent)
	CID_IntentMove, CID_IntentCast ComponentID
)

// RenderComponent — Визуал.
// Храним упакованный Glyph (4 байта) вместо строк.
type RenderComponent struct {
	ecs.StateMarker
	Glyph glyph.Glyph
}

// PositionComponent — где находится сущность.
// Дискретная сетка (Tile-based).
type PositionComponent struct {
	ecs.StateMarker
	grid.TilePos // Анонимное поле: методы InRadius/Distance доступны напрямую!
}

// StatsComponent — ХП, Мана, Сила.
type StatsComponent struct {
	ecs.StateMarker
	IsDead    bool
	Health    int32
	MaxHealth int32
	Mana      int32
	MaxMana   int32
}

// NameComponent — Имя для логов и клиента.
type NameComponent struct {
	ecs.StateMarker
	Name string
}

// SpellbookComponent хранит состояние книги заклинаний сущности.
type SpellbookComponent struct {
	ecs.StateMarker
	KnownSpells []uint32           // ID спеллов из справочника
	Cooldowns   map[uint32]float64 // ID -> Время, когда спелл откатится (в секундах или тиках)
	GCD         float64            // Timestamp окончания Глобального КД
}

// ControllerComponent — Связь с внешним миром.
// Если этот компонент есть — сущность управляется Агентом (человеком или AI-ботом через сеть).
type ControllerComponent struct {
	ecs.StateMarker
	AgentID string // ID сессии / Сокета / LLM Context ID
}
