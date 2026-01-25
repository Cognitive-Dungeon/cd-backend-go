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
	pathMap       = flag.String("map", "tools/tiled/demo_map.tmj", "map file")
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
		if m.Flags.Has(worldmap.FlagSolid) {
			vis.Style = lipgloss.NewStyle().Foreground(vis.Color).Bold(true)
		} else {
			vis.Style = lipgloss.NewStyle().Foreground(vis.Color)
		}
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
	loadedChunks int
	memStats     runtime.MemStats
	fps          float64
	lastFrame    time.Time

	debugGrid bool
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
	_, _ = loader.LoadRegionChunkCenter(geo.Pos(0, 0, 0), 2)

	// auto center camera
	minB, maxB := w.Bounds()
	camX := ((minB.X() + maxB.X()) * worldmap.ChunkSize) / 2
	camY := ((minB.Y() + maxB.Y()) * worldmap.ChunkSize) / 2

	return model{world: w, visuals: vis, camX: camX, camY: camY, debugGrid: true}
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
		count := 0
		m.world.RangeChunks(func(_ geo.Location, _ *worldmap.Chunk) bool { count++; return true })
		m.loadedChunks = count

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
		switch strings.ToLower(msg.String()) {
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
		}
	}
	return m, nil
}

// ---------------- Render ----------------

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	sidebarWidth := 40
	mapW := m.width - sidebarWidth - 4
	mapH := m.height - 4

	if mapW < 20 {
		mapW = 20
	}
	if mapH < 10 {
		mapH = 10
	}

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
			info := m.world.GetInfo(pos)

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
				switch {
				case info.Flags.Has(worldmap.FlagSolid):
					ch = "█"
				case info.Flags.Has(worldmap.FlagLiquid):
					ch = "~"
				case info.Flags.Has(worldmap.FlagWalkable):
					ch = "."
				default:
					ch = " "
				}
			}

			style := vis.Style

			if m.debugGrid {

				if wx%worldmap.ChunkSize == 0 {
					style = style.Foreground(lipgloss.Color("#5555FF"))
					ch = "|"
				}
				if wy%worldmap.ChunkSize == 0 {
					style = style.Foreground(lipgloss.Color("#5555FF"))
					ch = "-"
				}
				if wx%worldmap.ChunkSize == 0 && wy%worldmap.ChunkSize == 0 {
					style = style.Foreground(lipgloss.Color("#5555FF"))
					ch = "+"
				}
			}

			if wx == m.camX && wy == m.camY {
				style = style.Background(lipgloss.Color("#ffffff")).Foreground(lipgloss.Color("#000000"))
			}

			sb.WriteString(style.Render(ch + " "))
		}
		if y < tilesY-1 {
			sb.WriteString("\n")
		}
	}

	mapView := lipgloss.NewStyle().Width(mapW).Height(mapH).Border(lipgloss.RoundedBorder()).Render(sb.String())

	// sidebar
	minB, maxB := m.world.Bounds()
	cursorPos := geo.Pos(m.camX, m.camY, 0)
	chunk := worldmap.GetChunkKey(cursorPos)
	lx, ly := worldmap.GetLocalCoords(cursorPos)
	info := m.world.GetInfo(cursorPos)
	matName := "Void"
	if v, ok := m.visuals.Defs[info.Material]; ok {
		matName = v.Name
	}

	sidebar := fmt.Sprintf(
		"World Bounds: %s -> %s\n"+
			"Camera: (%d,%d)\nChunk: %s  Local:(%d,%d)\n\n"+
			"Chunks Loaded: %d\nFPS: %.1f\nMem: %.1f MB\nGC: %d\n\n"+
			"--- Tile ---\n"+"Material: %d (%s)\n"+"Flags: %s\n\n"+
			"[WASD] move tile\n[IJKL] jump chunk\n[G] toggle grid\n",
		minB.String(), maxB.String(),
		m.camX, m.camY,
		chunk.String(), lx, ly,
		m.loadedChunks,
		m.fps,
		float64(m.memStats.Alloc)/1024/1024,
		m.memStats.NumGC,
		info.Material, matName,
		flagsToString(info.Flags),
	)

	sidebarView := lipgloss.NewStyle().Width(sidebarWidth).Padding(1).Border(lipgloss.RoundedBorder()).Render(sidebar)

	status := fmt.Sprintf("Tile:%s | Chunk:%s | Chunks:%d | FPS:%.1f | Heap:%.1fMB", cursorPos.String(), chunk.String(), m.loadedChunks, m.fps, float64(m.memStats.Alloc)/1024/1024)
	statusBar := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(status)

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
	if f.Has(worldmap.FlagWalkable) {
		s = append(s, "WALK")
	}
	if f.Has(worldmap.FlagLiquid) {
		s = append(s, "LIQUID")
	}
	if f.Has(worldmap.FlagOpaque) {
		s = append(s, "OPAQUE")
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
