package version

import (
	"fmt"
	"runtime"
)

// Переменные, которые заполняет linker (Mage)
// Значения по умолчанию - на случай запуска через `go run main.go` без флагов
var (
	Tag       = "v0.0.0" // Семантическая версия
	BuildNum  = "0"      // Дней с эпохи (строка, т.к. -X принимает строки, но Mage пошлет число)
	Commit    = "unknown"
	Branch    = "unknown"
	DirtyStr  = "false" // "true" или "false"
	BuildTime = "unknown"
)

// Info структура для API
type Info struct {
	Version   string `json:"version"`   // v0.1.0
	Build     string `json:"build_num"` // 3890
	Commit    string `json:"commit"`
	Branch    string `json:"branch"`
	IsDirty   bool   `json:"is_dirty"`
	BuildTime string `json:"build_time"`
	GoVersion string `json:"go_version"`
}

// Get возвращает структуру
func Get() Info {
	return Info{
		Version:   Tag,
		Build:     BuildNum,
		Commit:    Commit,
		Branch:    Branch,
		IsDirty:   DirtyStr == "true",
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
	}
}

func String() string {
	dirtyMark := ""
	if DirtyStr == "true" {
		dirtyMark = " (dirty)"
	}

	// Формат: "Cognitive Server v0.1.0 Build 58 [a1b2c] (dirty)"
	return fmt.Sprintf(
		"%s Build %s [%s]%s",
		Tag,       // v1.0.2
		BuildNum,  // 58
		Commit,    // a1b2c
		dirtyMark, // (dirty) или пусто
	)
}

// FullString для подробного лога при старте
func FullString() string {
	return fmt.Sprintf(
		"Cognitive Server %s\n Branch: %s\n Built: %s",
		String(),
		Branch,
		BuildTime,
	)
}
