package engine

// SpellEffectType — что делает заклинание.
type SpellEffectType int

const (
	EffectDummy    SpellEffectType = iota
	EffectDamage                   // Нанести урон
	EffectHeal                     // Полечить
	EffectTeleport                 // Скачок (Blink)
)

// SpellInfo — "Паспорт" заклинания (Read-only static data).
type SpellInfo struct {
	ID        uint32
	Name      string
	CastTime  float64 // Секунды. 0 = Instant
	Cooldown  float64 // Секунды
	Range     int     // Дистанция в клетках
	Effect    SpellEffectType
	BaseValue int // Сила эффекта (урон/хил)
	Cost      int // Цена маны
}

// SpellRegistry — Глобальный справочник всех заклинаний игры.
var SpellRegistry = map[uint32]SpellInfo{
	// 1. Обычная атака (Melee)
	1: {
		ID: 1, Name: "Attack",
		CastTime: 0, Cooldown: 1.5, Range: 1,
		Effect: EffectDamage, BaseValue: 10, Cost: 0,
	},
	// 2. Огненный шар
	2: {
		ID: 2, Name: "Fireball",
		CastTime: 2.0, Cooldown: 0, Range: 10,
		Effect: EffectDamage, BaseValue: 30, Cost: 20,
	},
	// 3. Малое лечение
	3: {
		ID: 3, Name: "Lesser Heal",
		CastTime: 1.5, Cooldown: 0, Range: 30,
		Effect: EffectHeal, BaseValue: 40, Cost: 15,
	},
	// 4. Скачок (Мгновенное перемещение)
	4: {
		ID: 4, Name: "Blink",
		CastTime: 0, Cooldown: 15.0, Range: 5,
		Effect: EffectTeleport, BaseValue: 0, Cost: 10,
	},
}
