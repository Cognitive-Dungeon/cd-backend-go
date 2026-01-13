package engine

import "cognitive-server/internal/core/types"

// RenderComponent — Визуал.
// Храним упакованный Glyph (4 байта) вместо строк.
type RenderComponent struct {
	Glyph types.Glyph
}

// PositionComponent — где находится сущность.
// Дискретная сетка (Tile-based).
type PositionComponent struct {
	types.TilePos
}

// StatsComponent — ХП, Мана, Сила.
type StatsComponent struct {
	IsDead    bool
	Health    int32
	MaxHealth int32
	Mana      int32
	MaxMana   int32
}

// NameComponent — Имя для логов и клиента.
type NameComponent struct {
	Name string
}

// SpellbookComponent — Что существо умеет кастовать.
type SpellbookComponent struct {
	KnownSpells []uint32           // ID спеллов из справочника
	Cooldowns   map[uint32]float64 // ID -> Время, когда спелл откатится (в секундах или тиках)
	GCD         float64            // Timestamp когда ограничение на каст спадёт
}

// ControllerComponent — Связь с внешним миром.
// Если этот компонент есть — сущность управляется Агентом (человеком или AI-ботом через сеть).
type ControllerComponent struct {
	AgentID string // ID сессии / Сокета / LLM Context ID
}
