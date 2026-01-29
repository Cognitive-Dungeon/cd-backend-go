package cognitive_builder

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/pterm/pterm"
)

// ToolBuilder — конфигуратор запуска утилиты
type ToolBuilder struct {
	toolName   string   // Имя бинарника (напр. "tileset_gen")
	toolSrcDir string   // Где лежат исходники (если это Go tool)
	inputs     []string // Файлы, от которых зависит результат (для кэша)
	args       []string // Аргументы запуска
	outputs    []string // Файлы, которые должны появиться (для проверки)
}

// Tool создает новый пайплайн запуска инструмента
func Tool(name string) *ToolBuilder {
	return &ToolBuilder{
		toolName: name,
	}
}

// FromSource указывает, где лежат исходники инструмента.
// Если указано, система сначала попробует собрать этот инструмент.
func (t *ToolBuilder) FromSource(path string) *ToolBuilder {
	t.toolSrcDir = path
	return t
}

// Input добавляет файлы/глобы в отслеживание изменений.
func (t *ToolBuilder) Input(patterns ...string) *ToolBuilder {
	t.inputs = append(t.inputs, patterns...)
	return t
}

// Output добавляет ожидаемые выходные файлы (проверка существования).
func (t *ToolBuilder) Output(paths ...string) *ToolBuilder {
	t.outputs = append(t.outputs, paths...)
	return t
}

// Arg добавляет аргументы командной строки.
func (t *ToolBuilder) Arg(args ...string) *ToolBuilder {
	t.args = append(t.args, args...)
	return t
}

// Run выполняет пайплайн: Build Tool -> Check Cache -> Run Tool
func (t *ToolBuilder) Run() error {
	exePath := ExePath(t.toolName)

	// ШАГ 1: Собираем сам инструмент (если это Go tool)
	if t.toolSrcDir != "" {
		err := Build(t.toolName).
			WithSrcDir(t.toolSrcDir).
			Exec()

		if err != nil {
			return fmt.Errorf("failed to build tool %s: %w", t.toolName, err)
		}
	}

	// ШАГ 2: Проверяем, нужно ли запускать генерацию
	// Хэш зависит от:
	// 1. Входных файлов (ассетов)
	// 2. Самого бинарника инструмента (если изменился код генератора)
	hashInputs := append(t.inputs, exePath)

	// Разрешаем глобы для инпутов
	filesToHash, err := resolveGlobs(hashInputs)
	if err != nil {
		return err
	}

	cache := loadCache()
	currentHash, err := calculateHashForFiles(filesToHash)
	if err != nil {
		return err
	}

	// Проверяем кэш и наличие выходных файлов
	allOutputsExist := true
	for _, out := range t.outputs {
		if _, err := os.Stat(out); os.IsNotExist(err) {
			allOutputsExist = false
			break
		}
	}

	taskName := "run:" + t.toolName + ":" + calculateArgsHash(t.args) // Уникальное имя для кэша

	if allOutputsExist && cache[taskName] == currentHash {
		LogSkip(t.toolName + " generation")
		return nil
	}

	// ШАГ 3: Запуск
	fmt.Println()
	LogHeader(fmt.Sprintf("Running Tool: %s", t.toolName))
	start := time.Now()

	// pterm style run
	err = runAndStream(exePath, t.args...)

	duration := time.Since(start)
	res := BuildResult{
		Name:     "run:" + t.toolName,
		OutPath:  "assets",
		Duration: duration,
	}

	if err != nil {
		res.Error = err
		Stats.Add(res)
		Notify("Tool Failed", t.toolName, true)
		return err
	}

	// ШАГ 4: Сохраняем кэш
	cache[taskName] = currentHash
	_ = cache.save()

	pterm.Success.Printf("Tool %s finished in %s\n", t.toolName, duration.Round(time.Millisecond))
	Stats.Add(res)
	return nil
}

// calculateArgsHash добавляет аргументы в уникальность ключа кэша,
// чтобы если вы поменяли флаги запуска, задача перезапустилась.
func calculateArgsHash(args []string) string {
	// Простая реализация, можно лучше
	s := fmt.Sprintf("%v", args)
	return fmt.Sprintf("%x", s)
}

// runAndStream запускает команду и печатает вывод с отступами и цветами
func runAndStream(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Читаем STDOUT (серый цвет)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			// Красивый отступ: "  │ строка вывода"
			pterm.FgYellow.Printf("  │ %s\n", scanner.Text())
		}
	}()

	// Читаем STDERR (красный цвет)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			pterm.FgRed.Printf("  │ %s\n", scanner.Text())
		}
	}()

	wg.Wait()
	return cmd.Wait()
}
