package data

import (
	"cognitive-server/internal/core/types"
	"encoding/json"
	"fmt"
	"os"
	"sync"
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
	// Ипользуем RWMutex на случай, если захотим делать Hot-Reload json-ов на лету.
	mu     sync.RWMutex
	spells map[types.SpellID]SpellDef
}

// NewSpellRegistry создает новый пустой реестр.
func NewSpellRegistry() *SpellRegistry {
	return &SpellRegistry{
		spells: make(map[types.SpellID]SpellDef),
	}
}

// LoadFromFile загружает данные из JSON файла.
// Возвращает ошибку, если файл не найден или JSON некорректен.
func (r *SpellRegistry) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read spell file %s: %w", path, err)
	}

	var spellList []SpellDef
	if err := json.Unmarshal(data, &spellList); err != nil {
		return fmt.Errorf("failed to parse spell json: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, spell := range spellList {
		r.spells[spell.ID] = spell
	}

	return nil
}

// Get возвращает определение заклинания по ID.
// Второй параметр (bool) указывает, найдено ли заклинание.
func (r *SpellRegistry) Get(id types.SpellID) (SpellDef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.spells[id]
	return s, ok
}
