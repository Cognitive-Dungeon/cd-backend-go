package engine

import (
	"cognitive-server/internal/core/types"
	"encoding/json"
	"os"
)

// SpellEffectDef описывает один эффект заклинания (урон, хил и т.д.)
type SpellEffectDef struct {
	Type      types.SpellEffectType `json:"type"`
	BaseValue int32                 `json:"baseValue"` // Базовое значение (урон)
	Target    string                `json:"target"`    // "TARGET_ENEMY", "TARGET_SELF"
}

// SpellDef — это "DBC запись". Статическое описание заклинания.
type SpellDef struct {
	ID          types.SpellID `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`

	// Характеристики
	Range      float64              `json:"range"` // Дистанция (в метрах/тайлах)
	Cooldown   float64              // Секунды
	Attributes types.SpellAttribute `json:"attributes"`

	// Ресурсы
	CostType  types.SpellResourceType `json:"costType"`
	CostValue int32                   `json:"costValue"`

	// Эффекты
	Effects []SpellEffectDef `json:"effects"`
}

// SpellRegistry хранит загруженные спеллы.
// Используем map для быстрого поиска по ID.
type SpellRegistry struct {
	spells map[types.SpellID]SpellDef
}

func NewSpellRegistry() *SpellRegistry {
	return &SpellRegistry{
		spells: make(map[types.SpellID]SpellDef),
	}
}

func (r *SpellRegistry) LoadFromFile(path string) error {
	// Открываем файл
	// В реальном проекте путь должен быть абсолютным или относительно бинарника
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var spellList []SpellDef
	if err := json.Unmarshal(data, &spellList); err != nil {
		return err
	}

	// Индексируем
	for _, spell := range spellList {
		r.spells[spell.ID] = spell
	}
	return nil
}

func (r *SpellRegistry) Get(id types.SpellID) (SpellDef, bool) {
	s, ok := r.spells[id]
	return s, ok
}
