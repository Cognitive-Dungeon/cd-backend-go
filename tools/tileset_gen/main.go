package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"cognitive-server/pkg/worldmap"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// --- CONFIG ---

const (
	TileSize = 32
	FontSize = 24.0
	DPI      = 72
)

// --- INPUT STRUCTURE (JSON) ---
// Структура, которую мы читаем из assets/tiles_def.json
type InputTileDef struct {
	Name    string   `json:"name"`
	ID      uint16   `json:"id"` // Мапится в MaterialID
	Symbol  string   `json:"symbol"`
	Color   string   `json:"color"`
	BgColor string   `json:"bg_color,omitempty"`
	Flags   []string `json:"flags"` // "solid", "opaque", "liquid"
	Desc    string   `json:"desc"`
	Variant uint8    `json:"variant,omitempty"`
}

// --- SERVER OUTPUT STRUCTURE ---
// Данные, которые загружает сервер при старте
type ServerTileMetadata struct {
	GID        int                 `json:"gid"` // Global ID (Tiled ID + 1)
	MaterialID worldmap.MaterialID `json:"material_id"`
	Flags      worldmap.TileFlag   `json:"flags"`
	Variant    uint8               `json:"variant"`
	Name       string              `json:"name"`
	LLMDesc    string              `json:"llm_desc"`
	// Fallback для ASCII клиентов
	Char  string `json:"char"`
	Color string `json:"color"`
}

// --- TILED TSJ OUTPUT STRUCTURES ---
// Структуры для создания tileset.tsj (формат Tiled JSON)

type TiledProperty struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"` // "bool", "string", "int", "color"
	Value interface{} `json:"value"`
}

type TiledTileInfo struct {
	ID         int             `json:"id"`   // Локальный ID в тайлсете (начинается с 0)
	Type       string          `json:"type"` // Класс/Тип тайла
	Properties []TiledProperty `json:"properties,omitempty"`
}

type TiledTileset struct {
	Name         string          `json:"name"`
	TileWidth    int             `json:"tilewidth"`
	TileHeight   int             `json:"tileheight"`
	TileCount    int             `json:"tilecount"`
	Columns      int             `json:"columns"`
	Image        string          `json:"image"`
	ImageWidth   int             `json:"imagewidth"`
	ImageHeight  int             `json:"imageheight"`
	Margin       int             `json:"margin"`
	Spacing      int             `json:"spacing"`
	Tiles        []TiledTileInfo `json:"tiles,omitempty"`
	Type         string          `json:"type"`         // "tileset"
	Version      string          `json:"version"`      // "1.10"
	TiledVersion string          `json:"tiledversion"` // "1.10.2"
}

func main() {
	// 1. Настройка флагов командной строки
	inputFile := flag.String("in", "assets/tiles_def.json", "Input definition JSON path")
	fontPath := flag.String("font", "tools/tileset_gen/unifont-17.0.03.ttf", "TrueType font path")

	// Выходные пути
	serverOutDir := flag.String("server-out-dir", "assets", "Server assets output directory")
	tiledOutDir := flag.String("tiled-out-dir", "tools/tiled/assets", "Tiled assets output directory")
	outTiledImgName := "fallback_tileset.png"
	outTiledTsjName := "fallback_tileset.tsj"
	outServerName := "materials.json"

	flag.Parse()

	// 2. Чтение входного JSON
	rawBytes, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Failed to read input JSON (%s): %v", *inputFile, err)
	}

	var inputDefs []InputTileDef
	if err := json.Unmarshal(rawBytes, &inputDefs); err != nil {
		log.Fatalf("Failed to parse input JSON: %v", err)
	}
	fmt.Printf("Loaded definitions: %d items\n", len(inputDefs))

	// 3. Загрузка шрифта
	fontBytes, err := os.ReadFile(*fontPath)
	if err != nil {
		log.Fatalf("Failed to read font (%s): %v", *fontPath, err)
	}
	f, err := truetype.Parse(fontBytes)
	if err != nil {
		log.Fatal(err)
	}

	// 4. Подготовка изображения
	// Рассчитываем размер сетки
	count := len(inputDefs)
	cols := 8 // Фиксируем ширину в 8 тайлов
	rows := int(math.Ceil(float64(count) / float64(cols)))

	width := cols * TileSize
	height := rows * TileSize

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Настройка рисовальщика шрифта
	face := truetype.NewFace(f, &truetype.Options{
		Size:    FontSize,
		DPI:     DPI,
		Hinting: font.HintingFull,
	})

	// Списки для результатов
	var serverMeta []ServerTileMetadata
	var tiledTiles []TiledTileInfo

	// GID (Global ID) в Tiled maps обычно начинается с 1 (0 = пустота).
	// Но внутри файла .tsj (tileset definition) ID начинаются с 0.
	gid := 1

	// 5. Основной цикл генерации
	for i, def := range inputDefs {
		// Координаты
		col := i % cols
		row := i / cols
		x, y := col*TileSize, row*TileSize

		// A. РИСОВАНИЕ
		// Фон
		if def.BgColor != "" {
			bgC := parseHex(def.BgColor)
			draw.Draw(img, image.Rect(x, y, x+TileSize, y+TileSize), &image.Uniform{C: bgC}, image.Point{}, draw.Src)
		}

		// Символ
		fgC := parseHex(def.Color)

		// Центрирование текста (грубое, но для моноширинных шрифтов работает)
		dotX := fixed.I(x) + fixed.I(TileSize)/2 - fixed.I(int(FontSize/3))
		yOffset := (TileSize - int(FontSize)) * 2 / 3
		dotY := fixed.I(y) + fixed.I(TileSize) - fixed.I(yOffset)

		charToDraw := "?"
		if len([]rune(def.Symbol)) > 0 {
			charToDraw = string([]rune(def.Symbol)[0])
		}

		d := &font.Drawer{
			Dst:  img,
			Src:  &image.Uniform{C: fgC},
			Face: face,
			Dot:  fixed.Point26_6{X: dotX, Y: dotY},
		}
		d.DrawString(charToDraw)

		// B. ЛОГИКА (Флаги)
		finalFlags := resolveFlags(def.Flags)

		// C. ДАННЫЕ ДЛЯ СЕРВЕРА
		serverMeta = append(serverMeta, ServerTileMetadata{
			GID:        gid, // ID на карте
			MaterialID: worldmap.MaterialID(def.ID),
			Flags:      finalFlags,
			Variant:    def.Variant,
			Name:       def.Name,
			LLMDesc:    def.Desc,
			Char:       def.Symbol,
			Color:      def.Color,
		})

		// D. ДАННЫЕ ДЛЯ TILED (.tsj)
		localID := i // ID внутри тайлсета (0..N)

		tiledTile := TiledTileInfo{
			ID:   localID,
			Type: def.Name,
			Properties: []TiledProperty{
				{Name: "material_id", Type: "int", Value: def.ID},
				{Name: "desc", Type: "string", Value: def.Desc},
			},
		}

		// Добавляем булевы флаги для удобства дизайнера (is_solid: true)
		for _, fl := range def.Flags {
			propName := "is_" + strings.ToLower(strings.TrimSpace(fl))
			tiledTile.Properties = append(tiledTile.Properties, TiledProperty{
				Name:  propName,
				Type:  "bool",
				Value: true,
			})
		}

		tiledTiles = append(tiledTiles, tiledTile)
		gid++
	}

	// 6. СОХРАНЕНИЕ

	// PNG
	imgOutPath := filepath.Join(*tiledOutDir, outTiledImgName)
	fImg, err := os.Create(imgOutPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(fImg, img); err != nil {
		log.Fatal(err)
	}
	fImg.Close()
	fmt.Printf("[OK] Generated Image: %s (%dx%d)\n", imgOutPath, width, height)

	// Server JSON
	serverOutPath := filepath.Join(*serverOutDir, outServerName)
	fServer, err := os.Create(serverOutPath)
	if err != nil {
		log.Fatal(err)
	}
	encS := json.NewEncoder(fServer)
	encS.SetIndent("", "  ")
	encS.Encode(serverMeta)
	fServer.Close()
	fmt.Printf("[OK] Generated Server Data: %s\n", serverOutPath)

	// Tiled TSJ
	tsjOutPath := filepath.Join(*tiledOutDir, outTiledTsjName)

	tiledData := TiledTileset{
		Name:         "CD_AutoGeneratedTileset",
		TileWidth:    TileSize,
		TileHeight:   TileSize,
		TileCount:    count,
		Columns:      cols,
		Image:        outTiledImgName, // Tiled ищет картинку относительно .tsj файла
		ImageWidth:   width,
		ImageHeight:  height,
		Margin:       0,
		Spacing:      0,
		Tiles:        tiledTiles,
		Type:         "tileset",
		Version:      "1.10",
		TiledVersion: "1.10.2",
	}

	fTsj, err := os.Create(tsjOutPath)
	if err != nil {
		log.Fatal(err)
	}
	encT := json.NewEncoder(fTsj)
	encT.SetIndent("", "  ")
	encT.Encode(tiledData)
	fTsj.Close()
	fmt.Printf("[OK] Generated Tiled Tileset: %s\n", tsjOutPath)
}

// --- HELPERS ---

// resolveFlags превращает строки "solid", "opaque" в битовую маску TileFlag
func resolveFlags(flagNames []string) worldmap.TileFlag {
	var mask worldmap.TileFlag = worldmap.FlagNone

	for _, name := range flagNames {
		key := strings.ToLower(strings.TrimSpace(name))
		switch key {
		case "solid":
			mask |= worldmap.FlagSolid
		case "opaque":
			mask |= worldmap.FlagOpaque
		case "liquid":
			mask |= worldmap.FlagLiquid
		case "walkable":
			mask |= worldmap.FlagWalkable
		default:
			// Можно добавить лог предупреждения, если флаг неизвестен
			log.Printf("Warning: Unknown flag '%s'", key)
		}
	}
	return mask
}

// parseHex парсит #RRGGBB или RRGGBB в color.RGBA
func parseHex(s string) color.RGBA {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return color.RGBA{0, 0, 0, 0} // Прозрачный при ошибке
	}
	var r, g, b uint8
	fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}
