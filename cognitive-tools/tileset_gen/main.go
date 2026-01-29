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

// ============================================================
// CONFIG
// ============================================================

const (
	TileSize = 32
	FontSize = 24.0
	DPI      = 72
	MaxCols  = 8
)

// ============================================================
// INPUT STRUCTURE
// ============================================================

type InputTileDef struct {
	Name     string   `json:"name"`
	Category string   `json:"category"`
	ID       uint16   `json:"id"`
	Symbol   string   `json:"symbol"`
	Color    string   `json:"color"`
	BgColor  string   `json:"bg_color,omitempty"`
	Flags    []string `json:"flags"`
	Desc     string   `json:"desc"`
	Variant  uint8    `json:"variant,omitempty"`
}

// ============================================================
// SERVER OUTPUT
// ============================================================

type ServerTileMetadata struct {
	GID        int                 `json:"gid"`
	MaterialID worldmap.MaterialID `json:"material_id"`
	Flags      worldmap.TileFlag   `json:"flags"`
	Variant    uint8               `json:"variant"`
	Name       string              `json:"name"`
	LLMDesc    string              `json:"llm_desc"`
	Char       string              `json:"char"`
	Color      string              `json:"color"`
}

// ============================================================
// TILED STRUCTURES
// ============================================================

type TiledProperty struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type TiledTileInfo struct {
	ID         int             `json:"id"`
	Type       string          `json:"type"`
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
	Type         string          `json:"type"`
	Version      string          `json:"version"`
	TiledVersion string          `json:"tiledversion"`
}

// ============================================================
// CONFIG / FLAGS
// ============================================================

type Config struct {
	InputFile    string
	FontPath     string
	ServerOutDir string
	TiledOutDir  string
}

func parseFlags() Config {
	in := flag.String("in", "assets/tiles_def.jsonc", "Input definition JSON / JSONC path")
	font := flag.String("font", "cognitive-tools/tileset_gen/unifont-17.0.03.ttf", "TrueType font path")
	serverDir := flag.String("server-out-dir", "assets", "Server assets output directory")
	tiledDir := flag.String("tiled-out-dir", "cognitive-tools/tiled/assets", "Tiled assets output directory")
	flag.Parse()

	return Config{
		InputFile:    *in,
		FontPath:     *font,
		ServerOutDir: *serverDir,
		TiledOutDir:  *tiledDir,
	}
}

// ============================================================
// MAIN
// ============================================================

func main() {
	cfg := parseFlags()

	defs := loadInputDefs(cfg.InputFile)
	face := loadFont(cfg.FontPath)

	// 1. Получаем строгий порядок категорий из входного файла
	orderedCategories := extractOrderedCategories(defs)
	grouped := groupByCategory(defs)

	var allServerMeta []ServerTileMetadata

	// Глобальный счетчик GID. В Tiled нумерация всегда начинается с 1.
	currentGlobalGID := 1

	fmt.Println("==================================================")
	fmt.Println("ПОРЯДОК ИМПОРТА В TILED (ВАЖНО!)")
	fmt.Println("Добавляйте тайлсеты в карту строго в этом порядке:")
	fmt.Println("==================================================")

	// 2. Итерируемся строго по порядку появления категорий
	for _, category := range orderedCategories {
		tiles := grouped[category]

		// --- A. Генерация тайлсета для Tiled ---
		img := renderCategoryPNG(tiles, face)
		imgName := fmt.Sprintf("%d_%s.png", currentGlobalGID, category)
		tsjName := fmt.Sprintf("%d_tileset_%s.tsj", currentGlobalGID, category)

		savePNG(cfg.TiledOutDir, imgName, img)

		b := img.Bounds()
		ts := generateTileset(category, tiles, imgName, b.Dx(), b.Dy())
		saveJSON(cfg.TiledOutDir, tsjName, ts)

		// Вывод инструкции для пользователя
		fmt.Printf("FirstGID: %-4d -> %s \t(%d тайлов)\n",
			currentGlobalGID, tsjName, len(tiles))

		// --- Б. Генерация метаданных для Сервера (materials.json) ---
		for _, def := range tiles {
			meta := ServerTileMetadata{
				GID:        currentGlobalGID,
				MaterialID: worldmap.MaterialID(def.ID),
				Flags:      resolveFlags(def.Flags),
				Variant:    def.Variant,
				Name:       def.Name,
				LLMDesc:    def.Desc,
				Char:       def.Symbol,
				Color:      def.Color,
			}
			allServerMeta = append(allServerMeta, meta)
			currentGlobalGID++
		}
	}
	fmt.Println("==================================================")

	// 3. Сохраняем materials.json
	saveJSON(cfg.ServerOutDir, "materials.json", allServerMeta)
	fmt.Printf("[OK] materials.json сгенерирован (%d записей)\n", len(allServerMeta))
}

// ============================================================
// INPUT
// ============================================================

func loadInputDefs(path string) []InputTileDef {
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	raw = ToJSON(raw)

	var defs []InputTileDef
	if err := json.Unmarshal(raw, &defs); err != nil {
		log.Fatal(err)
	}
	return defs
}

// ============================================================
// FONT
// ============================================================

func loadFont(path string) font.Face {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	f, err := truetype.Parse(data)
	if err != nil {
		log.Fatal(err)
	}
	return truetype.NewFace(f, &truetype.Options{
		Size: FontSize,
		DPI:  DPI,
	})
}

// ============================================================
// GROUPING
// ============================================================

func groupByCategory(defs []InputTileDef) map[string][]InputTileDef {
	out := make(map[string][]InputTileDef)
	for _, def := range defs {
		out[def.Category] = append(out[def.Category], def)
	}
	return out
}

// extractOrderedCategories возвращает список категорий в том порядке,
// в котором они впервые встречаются в json файле.
func extractOrderedCategories(defs []InputTileDef) []string {
	seen := make(map[string]bool)
	var order []string
	for _, def := range defs {
		if !seen[def.Category] {
			seen[def.Category] = true
			order = append(order, def.Category)
		}
	}
	return order
}

// ============================================================
// PNG RENDERING
// ============================================================

func renderCategoryPNG(defs []InputTileDef, face font.Face) *image.RGBA {
	cols := min(len(defs), MaxCols)
	rows := int(math.Ceil(float64(len(defs)) / float64(cols)))

	img := image.NewRGBA(image.Rect(
		0, 0,
		cols*TileSize,
		rows*TileSize,
	))

	for i, def := range defs {
		x := (i % cols) * TileSize
		y := (i / cols) * TileSize
		rect := image.Rect(x, y, x+TileSize, y+TileSize)

		if def.BgColor != "" {
			draw.Draw(img, rect, &image.Uniform{parseHex(def.BgColor)}, image.Point{}, draw.Src)
		}

		drawSymbol(img, rect, def, face)
	}

	return img
}

func drawSymbol(
	img *image.RGBA,
	rect image.Rectangle,
	def InputTileDef,
	face font.Face,
) {
	runes := []rune(def.Symbol)
	if len(runes) == 0 {
		return
	}
	r := runes[0]

	// Цвет
	fg := parseHex(def.Color)

	// Измеряем глиф
	bounds, _, _ := face.GlyphBounds(r)

	glyphW := (bounds.Max.X - bounds.Min.X).Ceil()
	glyphH := (bounds.Max.Y - bounds.Min.Y).Ceil()

	// Центр тайла
	tileCX := rect.Min.X + TileSize/2
	tileCY := rect.Min.Y + TileSize/2

	// Смещение: центрируем глиф
	x := tileCX - glyphW/2 - bounds.Min.X.Ceil()
	y := tileCY + glyphH/2 - bounds.Max.Y.Ceil()

	d := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{fg},
		Face: face,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(string(r))
}

// ============================================================
// TILED TILESET
// ============================================================

func generateTileset(
	category string,
	defs []InputTileDef,
	imageName string,
	imageW int,
	imageH int,
) TiledTileset {

	tiles := make([]TiledTileInfo, 0, len(defs))
	for i, def := range defs {
		tiles = append(tiles, TiledTileInfo{
			ID:   i,
			Type: category,
			Properties: []TiledProperty{
				{Name: "name", Type: "string", Value: def.Name},
				{Name: "material_id", Type: "int", Value: def.ID},
				{Name: "desc", Type: "string", Value: def.Desc},
			},
		})
	}

	return TiledTileset{
		Name:         humanCategoryName(category),
		TileWidth:    TileSize,
		TileHeight:   TileSize,
		TileCount:    len(defs),
		Columns:      min(len(defs), MaxCols),
		Image:        imageName,
		ImageWidth:   imageW,
		ImageHeight:  imageH,
		Margin:       0,
		Spacing:      0,
		Tiles:        tiles,
		Type:         "tileset",
		Version:      "1.10",
		TiledVersion: "1.10.2",
	}
}

// ============================================================
// SERVER DATA
// ============================================================

func generateServerMetadata(defs []InputTileDef) []ServerTileMetadata {
	var out []ServerTileMetadata
	gid := 1

	for _, def := range defs {
		out = append(out, ServerTileMetadata{
			GID:        gid,
			MaterialID: worldmap.MaterialID(def.ID),
			Flags:      resolveFlags(def.Flags),
			Variant:    def.Variant,
			Name:       def.Name,
			LLMDesc:    def.Desc,
			Char:       def.Symbol,
			Color:      def.Color,
		})
		gid++
	}
	return out
}

// ============================================================
// HELPERS
// ============================================================

func savePNG(dir, name string, img image.Image) {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	png.Encode(f, img)
}

func saveJSON(dir, name string, v interface{}) {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func humanCategoryName(cat string) string {
	return strings.Title(cat)
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
