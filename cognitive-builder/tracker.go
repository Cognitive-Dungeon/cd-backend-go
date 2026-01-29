package cognitive_builder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pterm/pterm"
)

// BuildResult хранит данные об одном таргете
type BuildResult struct {
	Name     string        `json:"name"`
	OutPath  string        `json:"out_path"`
	Duration time.Duration `json:"duration_ns"` // В JSON уйдет как наносекунды
	Cached   bool          `json:"cached"`
	Error    error         `json:"-"` // Ошибки в JSON не пишем
}

// Tracker собирает статистику
type Tracker struct {
	mu      sync.Mutex
	Results []BuildResult
	Start   time.Time
}

var Stats = &Tracker{Start: time.Now()}

func (t *Tracker) Add(res BuildResult) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Results = append(t.Results, res)
}

// PrintSummary выводит красивую таблицу в конце
func (t *Tracker) PrintSummary() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.Results) == 0 {
		return
	}

	LogHeader("Build Summary")

	totalDuration := time.Since(t.Start)

	// Данные для таблицы
	tableData := [][]string{
		{"Target", "Status", "Duration", "Output"},
	}

	for _, res := range t.Results {
		status := pterm.FgGreen.Sprint("BUILT")
		dur := res.Duration.Round(time.Millisecond).String()

		if res.Cached {
			status = pterm.FgYellow.Sprint("CACHED")
			dur = "0ms"
		}
		if res.Error != nil {
			status = pterm.FgRed.Sprint("FAILED")
		}

		tableData = append(tableData, []string{
			res.Name, status, dur, res.OutPath,
		})
	}

	// Рендер таблицы
	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()

	pterm.Info.Printf("Total time: %s\n", totalDuration.Round(time.Millisecond))
}

// WriteManifest сохраняет JSON для CI/CD
func (t *Tracker) WriteManifest() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := ensureDirs(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t.Results, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(Dirs.Build, "manifest.json"), data, 0644)
}
