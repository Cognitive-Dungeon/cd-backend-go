package types

import (
	"encoding/json"
	"fmt"
	"strings"
)

// SpellAttribute — битовая маска свойств (аналог SpellFamilyFlags / Attributes в WoW).
type SpellAttribute uint32

const (
	SpellAttrNone SpellAttribute = 0

	// Поведенческие флаги
	SpellAttrChanneled       SpellAttribute = 1 << iota // Потоковое (нужно стоять)
	SpellAttrPassive                                    // Пассивное (нельзя скастовать)
	SpellAttrCastWhileMoving                            // Можно на бегу
	SpellAttrImpossible                                 // Нельзя сбить уроном
	SpellAttrInstant                                    // Мгновенное (хотя это дублируется CastTime=0, часто используют флаг)

	// Флаги целей
	SpellAttrTargetEnemy  // Только враги
	SpellAttrTargetFriend // Только друзья
	SpellAttrTargetSelf   // Только на себя

	// Флаги боя
	SpellAttrCantMiss // Не может промахнуться
	SpellAttrOffGCD   // Не запускает ГКД
)

// Has проверяет наличие флага
func (mask SpellAttribute) Has(flag SpellAttribute) bool {
	return mask&flag != 0
}

func (mask *SpellAttribute) Add(flag SpellAttribute) {
	*mask |= flag
}

func (mask *SpellAttribute) Remove(flag SpellAttribute) {
	*mask &^= flag
}

// Маппинг для JSON (Строка -> Бит)
var spellAttrMap = map[string]SpellAttribute{
	"CHANNELED":         SpellAttrChanneled,
	"PASSIVE":           SpellAttrPassive,
	"CAST_WHILE_MOVING": SpellAttrCastWhileMoving,
	"IMPOSSIBLE":        SpellAttrImpossible,
	"INSTANT":           SpellAttrInstant,
	"TARGET_ENEMY":      SpellAttrTargetEnemy,
	"TARGET_FRIEND":     SpellAttrTargetFriend,
	"TARGET_SELF":       SpellAttrTargetSelf,
	"CANT_MISS":         SpellAttrCantMiss,
	"OFF_GCD":           SpellAttrOffGCD,
}

// UnmarshalJSON позволяет писать в JSON ["CHANNELED", "TARGET_ENEMY"],
// а в памяти получать uint32.
func (mask *SpellAttribute) UnmarshalJSON(data []byte) error {
	var flags []string
	if err := json.Unmarshal(data, &flags); err != nil {
		return err
	}

	var result SpellAttribute
	for _, f := range flags {
		bit, ok := spellAttrMap[strings.ToUpper(f)]
		if !ok {
			return fmt.Errorf("unknown spell attribute: %s", f)
		}
		result |= bit
	}

	*mask = result
	return nil
}
