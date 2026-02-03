package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/worldmap"
	"cognitive-server/pkg/worldmap/storage/tiled"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ---------------- Config ----------------

var (
	pathMaterials = flag.String("materials", "assets/materials.json", "materials.json")
	pathMap       = flag.String("map", "cognitive-tools/tiled/demo_map.tmj", "map file")
)

// ---------------- Visuals ----------------

type MatVisual struct {
	ID    worldmap.MaterialID
	Name  string
	Char  string
	Color lipgloss.Color
	Style lipgloss.Style
}

type VisualsRegistry struct {
	Defs    map[worldmap.MaterialID]MatVisual
	Void    MatVisual
	Unknown MatVisual
}

func loadVisuals(path string) (*VisualsRegistry, tiled.Palette, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	type matJson struct {
		GID        int                 `json:"gid"`
		MaterialID worldmap.MaterialID `json:"material_id"`
		Flags      worldmap.TileFlag   `json:"flags"`
		Variant    uint8               `json:"variant"`
		Name       string              `json:"name"`
		Char       string              `json:"char"`
		Color      string              `json:"color"`
	}

	var raw []matJson
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, nil, err
	}

	reg := &VisualsRegistry{
		Defs:    map[worldmap.MaterialID]MatVisual{},
		Void:    MatVisual{Name: "VOID", Char: " ", Style: lipgloss.NewStyle().Background(lipgloss.Color("#000000"))},
		Unknown: MatVisual{Name: "???", Char: "?", Style: lipgloss.NewStyle().Foreground(lipgloss.Color("#FF00FF"))},
	}

	palette := tiled.Palette{}

	for _, m := range raw {
		vis := MatVisual{ID: m.MaterialID, Name: m.Name, Char: m.Char, Color: lipgloss.Color(m.Color)}
		// Стилизация: Жирный для твердых объектов
		style := lipgloss.NewStyle().Foreground(vis.Color)
		if m.Flags.Has(worldmap.FlagSolid) {
			style = style.Bold(true)
		}
		vis.Style = style
		reg.Defs[m.MaterialID] = vis

		palette[m.GID] = tiled.TileDef{MaterialID: m.MaterialID, Flags: m.Flags, Variant: m.Variant}
	}

	return reg, palette, nil
}

// ---------------- Model ----------------

type model struct {
	world   *worldmap.World
	visuals *VisualsRegistry

	width, height int
	camX, camY    int

	// metrics
	loadedStaticChunks int
	memStats           runtime.MemStats
	fps                float64
	lastFrame          time.Time

	debugGrid bool
	statusMsg string // Сообщение о действии (например, "Tile Changed")
}

// ---------------- Init ----------------

func initialModel() model {
	flag.Parse()

	vis, palette, err := loadVisuals(*pathMaterials)
	if err != nil {
		log.Fatal(err)
	}

	w := worldmap.NewWorld()
	store, err := tiled.NewStore(*pathMap, palette, geo.Pos(0, 0, 0))
	if err != nil {
		log.Fatal(err)
	}

	loader := worldmap.NewLoader(w, store)
	// Грузим область 2x2 чанка вокруг 0,0
	loaded, _ := loader.LoadRegionChunkCenter(geo.Pos(0, 0, 0), 4)
	fmt.Printf("Loaded %d static chunks initially\n", loaded)

	// Авто-центровка камеры
	var minX, maxX, minY, maxY int
	hasContent := false
	w.RangeChunks(func(pos geo.Location, _ *worldmap.Chunk) bool {
		cx, cy, _ := pos.XYZ()

		if !hasContent {
			// Инициализируем первым найденным значением
			minX, maxX = cx, cx
			minY, maxY = cy, cy
			hasContent = true
		} else {
			// Расширяем границы
			if cx < minX {
				minX = cx
			}
			if cx > maxX {
				maxX = cx
			}
			if cy < minY {
				minY = cy
			}
			if cy > maxY {
				maxY = cy
			}
		}
		return true
	})

	camX, camY := 0, 0
	if hasContent {
		// Центрируем камеру
		// Умножаем на ChunkSize, чтобы получить пиксели/тайлы
		camX = ((minX + maxX + 1) * worldmap.ChunkSize) / 2
		camY = ((minY + maxY + 1) * worldmap.ChunkSize) / 2
	} else {
		// Если мир пуст, встаем в 0,0
		fmt.Println("Warning: No chunks found to center camera")
	}

	return model{
		world:     w,
		visuals:   vis,
		camX:      camX,
		camY:      camY,
		debugGrid: true,
	}
}

// ---------------- BubbleTea ----------------

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second/60, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		runtime.ReadMemStats(&m.memStats)

		// Считаем статические чанки (Regions)
		count := 0
		m.world.RangeChunks(func(_ geo.Location, _ *worldmap.Chunk) bool { count++; return true })
		m.loadedStaticChunks = count

		now := time.Now()
		if !m.lastFrame.IsZero() {
			m.fps = 1.0 / now.Sub(m.lastFrame).Seconds()
		}
		m.lastFrame = now

		return m, tea.Tick(time.Second/60, func(t time.Time) tea.Msg { return tickMsg(t) })

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		step := 1
		if strings.ToUpper(msg.String()) == msg.String() {
			step = 5
		}
		key := strings.ToLower(msg.String())

		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "w", "up":
			m.camY -= step
		case "s", "down":
			m.camY += step
		case "a", "left":
			m.camX -= step
		case "d", "right":
			m.camX += step

		// Навигация по чанкам
		case "i":
			m.camY -= worldmap.ChunkSize
		case "k":
			m.camY += worldmap.ChunkSize
		case "j":
			m.camX -= worldmap.ChunkSize
		case "l":
			m.camX += worldmap.ChunkSize

		case "g":
			m.debugGrid = !m.debugGrid

		// --- NEW: Test Dynamic Layer ---
		case "e": // Edit (Toggle Wall)
			pos := geo.Pos(m.camX, m.camY, 0)
			current := m.world.GetTile(pos)

			// Меняем Пол <-> Стена
			newTile := worldmap.Tile{Material: 1, Flags: worldmap.FlagWalkable} // Floor
			if current.Material == 1 {
				newTile = worldmap.Tile{Material: 10, Flags: worldmap.FlagSolid | worldmap.FlagOpaque} // Wall
			}

			// Это вызовет создание SparseChunk и запись в Шард
			m.world.SetTile(pos, newTile)
			m.statusMsg = fmt.Sprintf("SetTile: %d -> %d", current.Material, newTile.Material)

		// --- NEW: Test Physics Cache ---
		case "c": // Check Collision
			pos := geo.Pos(m.camX, m.camY, 0)
			isSolid := m.world.IsSolidFast(pos)
			m.statusMsg = fmt.Sprintf("IsSolidFast(%s): %v", pos, isSolid)
		}
	}
	return m, nil
}

// ---------------- Render ----------------

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebarWidth := 45
	mapW := m.width - sidebarWidth - 4
	mapH := m.height - 4
	if mapW < 20 {
		mapW = 20
	}
	if mapH < 10 {
		mapH = 10
	}

	// Render Map
	tilesX := mapW / 2
	tilesY := mapH
	startX := m.camX - tilesX/2
	startY := m.camY - tilesY/2

	var sb strings.Builder

	for y := 0; y < tilesY; y++ {
		for x := 0; x < tilesX; x++ {
			wx := startX + x
			wy := startY + y
			pos := geo.Pos(wx, wy, 0)

			// 1. Get Info (Static + Dynamic overlay)
			info := m.world.GetInfo(pos)

			// 2. Resolve Visual
			vis := m.visuals.Void
			if info.Material != 0 {
				if v, ok := m.visuals.Defs[info.Material]; ok {
					vis = v
				} else {
					vis = m.visuals.Unknown
				}
			}

			ch := vis.Char
			if ch == "" {
				// Fallback char if materials.json missing char
				if info.Flags.Has(worldmap.FlagSolid) {
					ch = "█"
				} else {
					ch = "."
				}
			}

			style := vis.Style

			// 3. Grid Overlay
			if m.debugGrid {
				isChunkBorderX := wx%worldmap.ChunkSize == 0
				isChunkBorderY := wy%worldmap.ChunkSize == 0

				if isChunkBorderX || isChunkBorderY {
					style = style.Foreground(lipgloss.Color("#444444"))
					if isChunkBorderX {
						ch = "|"
					}
					if isChunkBorderY {
						ch = "-"
					}
					if isChunkBorderX && isChunkBorderY {
						ch = "+"
					}
				}
			}

			// 4. Cursor
			if wx == m.camX && wy == m.camY {
				style = style.Background(lipgloss.Color("#FFFFFF")).Foreground(lipgloss.Color("#000000"))
			}

			sb.WriteString(style.Render(ch + " "))
		}
		if y < tilesY-1 {
			sb.WriteString("\n")
		}
	}

	mapView := lipgloss.NewStyle().
		Width(mapW).Height(mapH).
		Border(lipgloss.RoundedBorder()).
		Render(sb.String())

	// --- Sidebar Logic ---
	cursorPos := geo.Pos(m.camX, m.camY, 0)
	chunkKey := worldmap.GetChunkKey(cursorPos)
	lx, ly := worldmap.GetLocalCoords(cursorPos)
	info := m.world.GetInfo(cursorPos)

	matName := "Void"
	if v, ok := m.visuals.Defs[info.Material]; ok {
		matName = v.Name
	}

	// Вычисляем архитектурные данные (дублируем логику world.go для отображения)
	cx, cy, _ := chunkKey.XYZ()
	// Region: 32x32 chunks
	regionX := cx >> 5
	regionY := cy >> 5
	// Shard: hash
	shardID := (cx ^ cy) & 63

	sidebar := fmt.Sprintf(
		"COORD:   %s\n"+
			"CHUNK:   %s  Local:(%d,%d)\n"+
			"REGION:  (%d, %d)\n"+
			"SHARD:   #%d\n\n"+
			"--- Architecture ---\n"+
			"Static Chunks: %d\n"+
			"FPS:           %.1f\n"+
			"Heap:          %.1f MB\n\n"+
			"--- Tile Info ---\n"+
			"Material: %d (%s)\n"+
			"Flags:    %s\n\n"+
			"--- Controls ---\n"+
			"[WASD] Move\n"+
			"[IJKL] Jump Chunk\n"+
			"[G]    Grid\n"+
			"[E]    Edit (Test Dynamic)\n"+
			"[C]    Check SolidFast\n\n"+
			"STATUS: %s",
		cursorPos.String(),
		chunkKey.String(), lx, ly,
		regionX, regionY,
		shardID,
		m.loadedStaticChunks,
		m.fps,
		float64(m.memStats.Alloc)/1024/1024,
		info.Material, matName,
		flagsToString(info.Flags),
		m.statusMsg,
	)

	sidebarView := lipgloss.NewStyle().
		Width(sidebarWidth).
		Padding(1).
		Border(lipgloss.RoundedBorder()).
		Render(sidebar)

	// Status Bar
	statusStr := fmt.Sprintf("Tile:%s | Shard:%d | Mat:%d", cursorPos.String(), shardID, info.Material)
	statusBar := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(statusStr)

	return lipgloss.JoinVertical(lipgloss.Top,
		lipgloss.JoinHorizontal(lipgloss.Top, mapView, sidebarView),
		statusBar,
	)
}

func flagsToString(f worldmap.TileFlag) string {
	var s []string
	if f.Has(worldmap.FlagSolid) {
		s = append(s, "SOLID")
	}
	if f.Has(worldmap.FlagOpaque) {
		s = append(s, "OPAQUE")
	}
	if f.Has(worldmap.FlagLiquid) {
		s = append(s, "LIQUID")
	}
	if f.Has(worldmap.FlagWalkable) {
		s = append(s, "WALK")
	}
	if len(s) == 0 {
		return "-"
	}
	return strings.Join(s, "|")
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
