package tiled

import (
	"cognitive-server/pkg/worldmap"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strconv"
)

// Config
const (
	Scale      = 8         // Сколько пикселей в одном тайле (8x8 квадратик)
	GridColor  = "#FF00FF" // Цвет сетки чанков (Маджента)
	EmptyColor = "#000000" // Цвет пустоты
)

// Visualizer управляет процессом отладочной отрисовки
type Visualizer struct {
	MatColors map[worldmap.MaterialID]color.RGBA
	OutputDir string
}

func NewVisualizer(outputDir string) *Visualizer {
	return &Visualizer{
		MatColors: make(map[worldmap.MaterialID]color.RGBA),
		OutputDir: outputDir,
	}
}

// LoadColors загружает маппинг MaterialID -> HexColor из materials.json
func (v *Visualizer) LoadColors(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Временная структура для чтения JSON
	type matColorDef struct {
		ID    worldmap.MaterialID `json:"material_id"`
		Color string              `json:"color"`
	}
	var mats []matColorDef
	if err := json.Unmarshal(data, &mats); err != nil {
		return err
	}

	for _, m := range mats {
		v.MatColors[m.ID] = parseHexColor(m.Color)
	}
	return nil
}

// DebugLayers сохраняет каждый слой Tiled как отдельное изображение
func (v *Visualizer) DebugLayers(m *Map, palette Palette) error {
	if err := os.MkdirAll(v.OutputDir, 0755); err != nil {
		return err
	}

	for i, layer := range m.Layers {
		if layer.Type != "tilelayer" {
			continue
		}

		// 1. Декодируем данные слоя
		var gids []uint32
		var err error

		// Обработка чанков (Infinite) или обычных слоев
		if len(layer.Chunks) > 0 {
			// Для простоты визуализации infinite слоев, соберем их на едином холсте,
			// вычислив границы. Это сложнее, поэтому для демо
			// просто пропустим сложную склейку infinite слоев в послойном просмотре
			// или реализуем упрощенно.
			fmt.Printf("Skipping infinite layer visualization for layer %d (use composite view)\n", i)
			continue
		} else {
			gids, err = decodeGIDData(layer.Data, layer.Compression)
		}

		if err != nil {
			return fmt.Errorf("layer %s decode error: %v", layer.Name, err)
		}

		// 2. Создаем изображение
		img := image.NewRGBA(image.Rect(0, 0, layer.Width*Scale, layer.Height*Scale))

		// 3. Заполняем пиксели
		for idx, gid := range gids {
			tileDef, ok := mapGid(gid, palette)
			col := parseHexColor(EmptyColor) // Default black

			if ok {
				if c, exists := v.MatColors[tileDef.Material]; exists {
					col = c
				}
			}

			// Рисуем квадрат Scale x Scale
			lx := idx % layer.Width
			ly := idx / layer.Width
			drawRect(img, lx*Scale, ly*Scale, Scale, col)
		}

		// 4. Сохраняем
		filename := fmt.Sprintf("01_layer_%02d_%s.png", i, layer.Name)
		if err := v.saveImage(img, filename); err != nil {
			return err
		}
	}
	return nil
}

// DebugComposite визуализирует итоговый Store (результат наложения) и сетку чанков
func (v *Visualizer) DebugComposite(store *Store) error {
	// 1. Вычисляем границы всего мира
	minX, minY, maxX, maxY := math.MaxInt, math.MaxInt, math.MinInt, math.MinInt

	// Проходим по всем чанкам
	for loc := range store.chunks {
		cx, cy, _ := loc.XYZ() // Координаты чанка

		// Переводим в тайловые координаты
		tx, ty := cx*worldmap.ChunkSize, cy*worldmap.ChunkSize

		if tx < minX {
			minX = tx
		}
		if ty < minY {
			minY = ty
		}
		if tx+worldmap.ChunkSize > maxX {
			maxX = tx + worldmap.ChunkSize
		}
		if ty+worldmap.ChunkSize > maxY {
			maxY = ty + worldmap.ChunkSize
		}
	}

	width := (maxX - minX)
	height := (maxY - minY)

	// Создаем холст
	img := image.NewRGBA(image.Rect(0, 0, width*Scale, height*Scale))

	// Цвет фона (для пустот между чанками)
	draw.Draw(img, img.Bounds(), &image.Uniform{parseHexColor("#111111")}, image.Point{}, draw.Src)

	// 2. Рисуем тайлы из Store
	for loc, chunk := range store.chunks {
		cx, cy, _ := loc.XYZ()
		chunkWorldX := cx * worldmap.ChunkSize
		chunkWorldY := cy * worldmap.ChunkSize

		for ly := 0; ly < worldmap.ChunkSize; ly++ {
			for lx := 0; lx < worldmap.ChunkSize; lx++ {
				tile, ok := chunk.GetTile(lx, ly)
				if !ok || tile.IsEmpty() {
					continue
				}

				col, ok := v.MatColors[tile.Material]
				if !ok {
					col = parseHexColor("#FF0000") // Error color (missing texture)
				}

				// Координаты на изображении (с учетом смещения мира)
				drawX := (chunkWorldX + lx - minX) * Scale
				drawY := (chunkWorldY + ly - minY) * Scale

				drawRect(img, drawX, drawY, Scale, col)
			}
		}
	}

	// 3. Рисуем сетку чанков (Grid)
	gridCol := parseHexColor(GridColor)
	// Вертикальные линии
	for x := 0; x <= width; x += worldmap.ChunkSize {
		drawVLine(img, x*Scale, gridCol)
	}
	// Горизонтальные линии
	for y := 0; y <= height; y += worldmap.ChunkSize {
		drawHLine(img, y*Scale, gridCol)
	}

	return v.saveImage(img, "02_final_composite_with_chunks.png")
}

// --- Helpers ---

func (v *Visualizer) saveImage(img image.Image, name string) error {
	path := filepath.Join(v.OutputDir, name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func drawRect(img *image.RGBA, x, y, size int, col color.RGBA) {
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			// Простейшая заливка, можно добавить border для тайлов
			if dx == size-1 || dy == size-1 {
				// Затемнение краев тайла (опционально)
				darker := col
				darker.R = uint8(float64(col.R) * 0.8)
				darker.G = uint8(float64(col.G) * 0.8)
				darker.B = uint8(float64(col.B) * 0.8)
				img.Set(x+dx, y+dy, darker)
			} else {
				img.Set(x+dx, y+dy, col)
			}
		}
	}
}

func drawVLine(img *image.RGBA, x int, col color.RGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		img.Set(x, y, col)
		img.Set(x+1, y, col) // Толщина 2px
	}
}

func drawHLine(img *image.RGBA, y int, col color.RGBA) {
	b := img.Bounds()
	for x := b.Min.X; x < b.Max.X; x++ {
		img.Set(x, y, col)
		img.Set(x, y+1, col) // Толщина 2px
	}
}

func parseHexColor(s string) color.RGBA {
	c := color.RGBA{A: 255}
	if len(s) == 0 {
		return c
	}
	if s[0] == '#' {
		s = s[1:]
	}
	hexVal, _ := strconv.ParseUint(s, 16, 32)

	if len(s) == 6 {
		c.R = uint8(hexVal >> 16)
		c.G = uint8((hexVal >> 8) & 0xFF)
		c.B = uint8(hexVal & 0xFF)
	} else if len(s) == 8 {
		// ARGB or RGBA parsing depends on format, assuming RGBA
		c.R = uint8(hexVal >> 24)
		c.G = uint8((hexVal >> 16) & 0xFF)
		c.B = uint8((hexVal >> 8) & 0xFF)
		c.A = uint8(hexVal & 0xFF)
	}
	return c
}
