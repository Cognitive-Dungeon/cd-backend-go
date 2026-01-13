package types

import (
	"encoding/json"
	"fmt"
)

// SpellID — уникальный идентификатор заклинания (как в WoW).
type SpellID uint32

// SpellResourceType — тип ресурса (HP, Mana, Energy).
type SpellResourceType uint8

const (
	SpellResourceNone SpellResourceType = iota
	SpellResourceHealth
	SpellResourceMana
	SpellResourceEnergy
)

// SpellSchool — школа магии (нужно для резистов в будущем).
type SpellSchool uint8

const (
	SpellSchoolPhysical SpellSchool = iota
	SpellSchoolHoly
	SpellSchoolFire
	SpellSchoolNature
	SpellSchoolFrost
	SpellSchoolShadow
	SpellSchoolArcane
)

// SpellEffectType — что делает конкретный эффект спелла.
type SpellEffectType uint8

const (
	SpellEffectNone SpellEffectType = iota
	SpellEffectSchoolDamage
	SpellEffectHeal
	SpellEffectTeleport
	SpellEffectApplyAura
)

var spellEffectTypeToString = map[SpellEffectType]string{
	SpellEffectNone:         "NONE",
	SpellEffectSchoolDamage: "SCHOOL_DAMAGE",
	SpellEffectHeal:         "HEAL",
	SpellEffectTeleport:     "TELEPORT",
	SpellEffectApplyAura:    "APPLY_AURA",
}

var spellEffectTypeFromString = map[string]SpellEffectType{
	"NONE":          SpellEffectNone,
	"SCHOOL_DAMAGE": SpellEffectSchoolDamage,
	"HEAL":          SpellEffectHeal,
	"TELEPORT":      SpellEffectTeleport,
	"APPLY_AURA":    SpellEffectApplyAura,
}

func (t SpellEffectType) String() string {
	if s, ok := spellEffectTypeToString[t]; ok {
		return s
	}
	return "UNKNOWN"
}

func (t SpellEffectType) MarshalJSON() ([]byte, error) {
	s, ok := spellEffectTypeToString[t]
	if !ok {
		return nil, fmt.Errorf("unknown SpellEffectType: %d", t)
	}
	return json.Marshal(s)
}

func (t *SpellEffectType) UnmarshalJSON(data []byte) error {
	// Попытка как строка
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if val, ok := spellEffectTypeFromString[s]; ok {
			*t = val
			return nil
		}
		return fmt.Errorf("unknown SpellEffectType: %q", s)
	}

	// Попытка как число
	var n uint8
	if err := json.Unmarshal(data, &n); err == nil {
		*t = SpellEffectType(n)
		return nil
	}

	return fmt.Errorf("invalid SpellEffectType: %s", string(data))
}
