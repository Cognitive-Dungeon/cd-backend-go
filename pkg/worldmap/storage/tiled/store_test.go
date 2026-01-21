package tiled

import (
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/worldmap"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// --- Константы ожидаемых материалов (из assets/materials.json) ---
const (
	MatPineTree    worldmap.MaterialID = 42 // GID 20
	MatRuinedWall  worldmap.MaterialID = 12 // GID 9
	MatWindowGlass worldmap.MaterialID = 24 // GID 15
)

// --- Вспомогательные структуры ---

type materialJSON struct {
	GID        int                 `json:"gid"`
	MaterialID worldmap.MaterialID `json:"material_id"`
	Flags      worldmap.TileFlag   `json:"flags"`
	Variant    uint8               `json:"variant"`
}

func loadRealPalette(t *testing.T, path string) Palette {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read materials.json at %s: %v", path, err)
	}

	var mats []materialJSON
	if err := json.Unmarshal(data, &mats); err != nil {
		t.Fatalf("failed to parse materials.json: %v", err)
	}

	p := make(Palette)
	for _, m := range mats {
		p[m.GID] = TileDef{
			MaterialID: m.MaterialID,
			Flags:      m.Flags,
			Variant:    m.Variant,
		}
	}
	return p
}

// mustGetTile скрывает сложность получения чанка и тайла.
func mustGetTile(t *testing.T, store *Store, globalPos geo.Location) worldmap.Tile {
	t.Helper()

	chunkKey := worldmap.GetChunkKey(globalPos)
	chunk, err := store.LoadChunk(chunkKey)
	if err != nil {
		t.Fatalf("❌ Ошибка загрузки чанка для позиции %s (ChunkKey: %s): %v", globalPos, chunkKey, err)
	}

	lx, ly := worldmap.GetLocalCoords(globalPos)
	tile, ok := chunk.GetTile(lx, ly)
	if !ok {
		t.Fatalf("❌ Координаты (%d, %d) выходят за пределы чанка %s", lx, ly, chunkKey)
	}

	return tile
}

// --- Основной тест ---

func TestNewStore_WithRealFiles(t *testing.T) {
	// 1. Настройка путей
	projectRoot := "../../../../"
	materialsPath := filepath.Join(projectRoot, "assets", "materials.json")
	mapPath := filepath.Join(projectRoot, "tools", "tiled", "demo_map.tmj")

	if _, err := os.Stat(mapPath); os.IsNotExist(err) {
		t.Skipf("Skipping integration test: file not found %s", mapPath)
	}

	// 2. Загрузка данных
	palette := loadRealPalette(t, materialsPath)
	store, err := NewStore(mapPath, palette, geo.Pos(0, 0, 0))
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// 3. Описание сценариев проверки
	tests := []struct {
		name        string              // Название сценария
		pos         geo.Location        // Глобальная координата проверки
		wantMat     worldmap.MaterialID // Ожидаемый материал
		wantSolid   bool                // Ожидаем ли мы, что тайл твердый?
		description string              // Пояснение
	}{
		{
			name:        "Ground Layer Check",
			pos:         geo.Pos(0, 0, 0),
			wantMat:     MatPineTree, // GID 20 -> 42
			wantSolid:   true,        // Флаг 3 (Solid | Opaque)
			description: "Проверяем базовый слой (Ground). В точке (0,0) должна быть сосна.",
		},
		{
			name:        "Walls Layer Overwrite",
			pos:         geo.Pos(25, 5, 0), // (Chunk 1, Local 9, 5)
			wantMat:     MatRuinedWall,     // GID 9 -> 12
			wantSolid:   true,
			description: "Проверяем наслоение. Слой Walls должен перекрыть слой Ground в точке (25,5).",
		},
		{
			name: "Transparent Overlay",
			pos:  geo.Pos(2, 2, 0),
			// В карте на этом месте GID 15 (Window Glass). Слой Walls здесь пуст.
			wantMat:     MatWindowGlass, // GID 15 -> 24
			wantSolid:   true,           // Window Glass имеет флаг Solid (1)
			description: "В слое Walls тут пусто, поэтому видим тайл с нижнего слоя Ground (Window Glass).",
		},
	}

	// 4. Запуск проверок
	fmt.Println("Запуск проверки контрольных точек карты...")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tile := mustGetTile(t, store, tt.pos)

			// Проверка материала
			if tile.Material != tt.wantMat {
				t.Errorf("Неверный материал в %s.\nОжидали: %d\nПолучили: %d\nСуть: %s",
					tt.pos, tt.wantMat, tile.Material, tt.description)
			}

			// Проверка флага проходимости
			isSolid := tile.Flags.Has(worldmap.FlagSolid)
			if isSolid != tt.wantSolid {
				t.Errorf("Неверный флаг Solid в %s.\nОжидали: %v\nПолучили: %v",
					tt.pos, tt.wantSolid, isSolid)
			}
		})
	}
}

func TestGenerateDebugImages(t *testing.T) {
	// Пути (как в прошлом тесте)
	projectRoot := "../../../../"
	materialsPath := filepath.Join(projectRoot, "assets", "materials.json")
	mapPath := filepath.Join(projectRoot, "tools", "tiled", "demo_map.tmj")
	outputDir := filepath.Join(projectRoot, "debug_visuals") // Папка для картинок

	if _, err := os.Stat(mapPath); os.IsNotExist(err) {
		t.Skip("Map file not found")
	}

	// 1. Инициализация визуализатора
	vis := NewVisualizer(outputDir)
	if err := vis.LoadColors(materialsPath); err != nil {
		t.Fatalf("Failed to load colors: %v", err)
	}

	// 2. Загрузка карты и палитры (реальный процесс)
	palette := loadRealPalette(t, materialsPath)

	// Читаем карту "сырой" для послойной визуализации
	rawMap, err := loadMapFile(mapPath)
	if err != nil {
		t.Fatalf("Failed to load raw map: %v", err)
	}

	// 3. Рендер Слоев (Визуализация входных данных)
	t.Log("Generating layer images...")
	if err := vis.DebugLayers(rawMap, palette); err != nil {
		t.Errorf("DebugLayers failed: %v", err)
	}

	// 4. Загрузка в Store (Обработка наслоений и чанков)
	t.Log("Loading map into Store...")
	store, err := NewStore(mapPath, palette, geo.Pos(0, 0, 0))
	if err != nil {
		t.Fatalf("Store creation failed: %v", err)
	}

	// 5. Рендер Финала (Визуализация памяти сервера)
	t.Log("Generating composite chunk map...")
	if err := vis.DebugComposite(store); err != nil {
		t.Errorf("DebugComposite failed: %v", err)
	}

	t.Logf("✅ Debug images saved to: %s", outputDir)
	t.Logf("Open '02_final_composite_with_chunks.png' to see the grid.")
}
