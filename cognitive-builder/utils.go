package cognitive_builder

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/magefile/mage/sh"
	"github.com/pterm/pterm"
)

// Dirs хранит все ключевые пути проекта
var Dirs = struct {
	Bin       string
	Build     string
	AssetsRaw string
	AssetsOut string
}{
	Bin:       "bin",
	Build:     "build",
	AssetsRaw: "raw_assets",
	AssetsOut: "assets",
}

const binDir = "bin"

// ensureDirs создает структуру папок перед сборкой
func ensureDirs() error {
	for _, dir := range []string{Dirs.Bin, Dirs.Build, Dirs.AssetsOut} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create dir %s: %w", dir, err)
		}
	}
	return nil
}

// ExePath возвращает путь к будущему бинарнику в папке bin
// Пример: "server" -> "bin/server.exe" (win) или "bin/server" (unix)
func ExePath(name string) string {
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(binDir, name)
}

// logStep выводит шаг сборки
func logStep(action, target string) {
	fmt.Printf("%s %s...\n", action, target)
}

// gitInfo возвращает результат git команды или "unknown"
func gitInfo(args ...string) string {
	out, err := sh.Output("git", args...)
	if err != nil {
		return "unknown"
	}
	return out
}

// envOrDefault читает переменную окружения или дефолт
func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func NamespaceToString(ns any) string {
	// 1. Если это уже строка, просто возвращаем её
	if name, ok := ns.(string); ok {
		return name
	}

	// 2. Иначе используем рефлексию для имени типа
	t := reflect.TypeOf(ns)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return strings.ToLower(t.Name())
}

func runGo(args ...string) error {
	fmt.Println("go", strings.Join(args, " "))
	return sh.Run("go", args...)
}

// Notify отправляет звуковой сигнал и заметное сообщение
func Notify(title, msg string, isError bool) {
	if isError {
		// 3 коротких писка при ошибке
		fmt.Print("\a\a\a")
		pterm.Error.Printfln("[%s] %s", title, msg)
	} else {
		// 1 писк при успехе
		fmt.Print("\a")
		pterm.Success.Printfln("[%s] %s", title, msg)
	}
}

func CleanAll() {
	LogHeader("Cleaning artifacts")
	cleanupDirs := []string{Dirs.Bin, Dirs.Build, Dirs.AssetsOut}

	for _, dir := range cleanupDirs {
		if err := os.RemoveAll(dir); err != nil {
			pterm.Error.Printf("Failed to remove %s: %v\n", dir, err)
		} else {
			pterm.Success.Printf("Removed %s\n", dir)
		}
	}
}
