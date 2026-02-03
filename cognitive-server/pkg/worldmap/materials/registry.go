package materials

import (
	"cognitive-server/pkg/types/glyph"
	"cognitive-server/pkg/worldmap"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// MaterialDef (Definition) — полное описание материала.
// Хранится в реестре.
type MaterialDef struct {
	Visual MaterialVisual
	Meta   *MaterialMeta
}

// MaterialVisual — "Горячие" данные для рендера (8 байт).
// Передаются по значению.
type MaterialVisual struct {
	ID    worldmap.MaterialID
	Glyph glyph.Glyph
}

// MaterialMeta — "Холодные" текстовые данные.
// Хранятся по указателю.
type MaterialMeta struct {
	Name        string
	Description string
}

type MaterialMetaProvider interface {
	GetMaterialMeta(id worldmap.MaterialID) *MaterialMeta
}

// MaterialRegistry — потокобезопасное хранилище определений.
type MaterialRegistry struct {
	visuals      []MaterialVisual
	metaProvider MaterialMetaProvider
	voidVisual   MaterialVisual
}

// NewRegistry создает пустой/дефолтный реестр (для тестов или фоллбека)
func NewRegistry() *MaterialRegistry {
	voidGlyph := glyph.MakeGlyph(0x000000, ' ')
	voidVisual := MaterialVisual{ID: 0, Glyph: voidGlyph}

	// Создаем минимальный слайс, чтобы доступ к ID=0 не падал
	visuals := make([]MaterialVisual, 1)
	visuals[0] = voidVisual

	return &MaterialRegistry{
		visuals: visuals,
		// Для пустого регистра провайдер возвращает заглушки
		metaProvider: &emptyProvider{},
		voidVisual:   voidVisual,
	}
}

// LoadRegistryFromFile загружает JSON (materials.json).
func LoadRegistryFromFile(path string) (*MaterialRegistry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open materials file: %w", err)
	}
	defer file.Close()

	// DTO для парсинга JSON
	type matRawDTO struct {
		MaterialID worldmap.MaterialID `json:"material_id"`
		Char       string              `json:"char"`
		Color      string              `json:"color"`
		// GID, Flags, Variant нам тут не нужны, они для физики/Tiled
	}

	var rawItems []json.RawMessage
	if err := json.NewDecoder(file).Decode(&rawItems); err != nil {
		return nil, err
	}

	// 1. Слайс визуальных представлений материалов
	// Ищем MaxID для аллокации
	// NOTE: materials.json состоит из разряженных объектов, рост id не постоянен
	maxID := worldmap.MaterialID(0)

	// Временная мапа для быстрого построения провайдера
	lazyMap := make(map[worldmap.MaterialID]json.RawMessage)

	for _, rawItem := range rawItems {
		// Парсим только ID и визуал
		var temp matRawDTO
		if err := json.Unmarshal(rawItem, &temp); err != nil {
			continue
		}

		if temp.MaterialID > maxID {
			maxID = temp.MaterialID
		}
		lazyMap[temp.MaterialID] = rawItem
	}

	// Аллоцируем слайс (Дырявый но быстрый)
	visuals := make([]MaterialVisual, int(maxID)+1)
	voidGlyph := glyph.MakeGlyph(0x000000, ' ')
	voidVisual := MaterialVisual{ID: 0, Glyph: voidGlyph}

	// Инициализируем слайс как void
	for i := range visuals {
		visuals[i] = voidVisual
	}

	// Заполняем визуалы из materials.json
	for id, raw := range lazyMap {
		var temp matRawDTO
		_ = json.Unmarshal(raw, &temp)
		g, err := glyph.ParseGlyphFromJSON(temp.Char, temp.Color)
		if err != nil {
			return nil, fmt.Errorf("failed to parse glyph for material %d: %w",
				temp.MaterialID, err)
		}
		visuals[id] = MaterialVisual{
			ID:    id,
			Glyph: g,
		}
	}
	return &MaterialRegistry{
		visuals:      visuals,
		metaProvider: newLazyJSONProvider(lazyMap),
		voidVisual:   voidVisual,
	}, nil
}

// GetVisual — Получить представление материала для визуализации.
// Возвращает структуру по значению (копирует 8 байт).
// Не трогает холодные данные (Meta).
// THREAD_SAFE
func (r *MaterialRegistry) GetVisual(id worldmap.MaterialID) MaterialVisual {
	if int(id) < len(r.visuals) {
		return r.visuals[id]
	}
	return r.voidVisual
}

// GetMeta — Получить метаданные материала.
func (r *MaterialRegistry) GetMeta(id worldmap.MaterialID) *MaterialMeta {
	return r.metaProvider.GetMaterialMeta(id)
}

// emptyProvider заглушка для NewRegistry
type emptyProvider struct{}

var emptyMeta = &MaterialMeta{Name: "Void", Description: "Nothing"}

func (e *emptyProvider) GetMaterialMeta(id worldmap.MaterialID) *MaterialMeta {
	return emptyMeta
}

type lazyJSONProvider struct {
	mu       sync.RWMutex
	rawStore map[worldmap.MaterialID]json.RawMessage
	cache    map[worldmap.MaterialID]*MaterialMeta
}

func newLazyJSONProvider(data map[worldmap.MaterialID]json.RawMessage) *lazyJSONProvider {
	return &lazyJSONProvider{
		rawStore: data,
		cache:    make(map[worldmap.MaterialID]*MaterialMeta),
	}
}

func (p *lazyJSONProvider) GetMaterialMeta(id worldmap.MaterialID) *MaterialMeta {
	// 1. Проверяем кэш
	p.mu.RLock()
	if m, ok := p.cache[id]; ok {
		p.mu.RUnlock()
		return m
	}

	// 2. Ищем сырые данные
	raw, exists := p.rawStore[id]
	p.mu.RUnlock()
	if !exists {
		return &MaterialMeta{Name: "Unknown", Description: "No data"}
	}

	// 3. Парсим "Тяжелые" данные
	// Структура только для метаданных
	type metaDTO struct {
		Name    string `json:"name"`
		LLMDesc string `json:"llm_desc"`
	}
	var dto metaDTO
	if err := json.Unmarshal(raw, &dto); err != nil {
		return &MaterialMeta{Name: "Error", Description: err.Error()}
	}

	// 4. Сохраняем в кэш
	meta := &MaterialMeta{
		Name:        dto.Name,
		Description: dto.LLMDesc,
	}
	p.mu.Lock()
	p.cache[id] = meta

	// Опционально: можно удалить raw[id] для экономии памяти,
	// если мы уверены, что GC обработает это эффективно.
	delete(p.rawStore, id)

	p.mu.Unlock()
	return meta
}
