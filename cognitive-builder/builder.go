package cognitive_builder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pterm/pterm"
)

// Builder предоставляет "текучий" (fluent) интерфейс для определения задач сборки.
type Builder struct {
	config TargetConfig
}

// Build начинает новую задачу сборки с настройками по умолчанию.
// Это точка входа, которая возвращает объект-строитель.
func Build(ns any) *Builder {
	name := NamespaceToString(ns)

	// Настройки по умолчанию
	config := TargetConfig{
		Name:       name,
		OutName:    name,
		SrcDir:     "",
		SrcGlobs:   []string{"go.mod", "go.sum"},
		WithMeta:   false,
		ModuleName: "",
	}

	return &Builder{config: config}
}

func (b *Builder) WithOutName(outName string) *Builder {
	b.config.OutName = outName
	return b
}

func (b *Builder) WithName(name string) *Builder {
	b.config.Name = name
	return b
}

// WithMeta включает вшивание метаданных версии.
func (b *Builder) WithMeta() *Builder {
	b.config.WithMeta = true
	return b
}

// WithModuleName позволяет переопределить имя Go модуля для ldflags.
func (b *Builder) WithModuleName(name string) *Builder {
	b.config.ModuleName = name
	return b
}

// WithSrcDir позволяет указать нестандартный путь к main.go.
func (b *Builder) WithSrcDir(path string) *Builder {
	b.config.SrcDir = path
	return b
}

// AddGlobs добавляет дополнительные паттерны файлов для отслеживания.
func (b *Builder) AddGlobs(globs ...string) *Builder {
	b.config.SrcGlobs = append(b.config.SrcGlobs, globs...)
	return b
}

// Exec - это терминальный метод. Он запускает сборку с настроенными параметрами.
// Всегда должен быть в конце цепочки.
func (b *Builder) Exec() error {
	if b.config.WithMeta && b.config.ModuleName == "" {
		return fmt.Errorf(
			"build target '%s': WithMeta() requires a ModuleName, but it was not set. Use WithModuleName() to specify it",
			b.config.Name,
		)
	}
	return SmartBuild(b.config)
}

// TargetConfig описывает задачу сборки
type TargetConfig struct {
	Name       string   // Имя таргета (для кэша и логов)
	SrcDir     string   // Где лежит main.go
	SrcGlobs   []string // Какие файлы отслеживать (напр. ["pkg/**/*.go", "go.mod"])
	OutName    string   // Имя бинарника
	ModuleName string   // Для ldflags
	WithMeta   bool     // Вшивать версию?
}

// SmartBuild — точка входа для сборки
func SmartBuild(opts TargetConfig) error {
	start := time.Now()
	res := BuildResult{Name: opts.Name, OutPath: ExePath(opts.OutName)}
	LogHeader(fmt.Sprintf("Building Target: %s", opts.Name))

	filesToHash, err := resolveGlobs(opts.SrcGlobs)
	if err != nil {
		return err
	}

	deps, err := GetTransitiveDeps(opts.SrcDir)
	if err != nil {
		return fmt.Errorf("failed to calculate dependencies: %w", err)
	}

	filesToHash = append(filesToHash, deps...)

	// 1. Проверяем кэш
	cache := loadCache()
	currentHash, err := calculateHashForFiles(filesToHash)
	if err != nil {
		return fmt.Errorf("hashing failed: %w", err)
	}

	// Если хэш совпадает и бинарник существует
	outPath := ExePath(opts.OutName)
	if _, err := os.Stat(outPath); err == nil && cache[opts.Name] == currentHash {
		LogSkip(opts.Name)
		res.Cached = true
		Stats.Add(res)
		return nil
	}

	// 2. Подготовка флагов
	args := []string{"build", "-trimpath", "-v", "-json"} // Включаем JSON output для парсинга

	if opts.WithMeta {
		v := getVersion()
		args = append(args, "-ldflags", v.LDFlags(opts.ModuleName))
	}

	args = append(args, "-o", outPath, "./"+opts.SrcDir)

	// 3. Запуск Go Build с парсингом вывода
	if err := runGoBuildWithProgress(args); err != nil {
		res.Duration = time.Since(start)
		res.Error = err
		Stats.Add(res)
		return err
	}
	res.Duration = time.Since(start)

	// 4. Обновление кэша
	cache[opts.Name] = currentHash
	if err := cache.save(); err != nil {
		log.Warn("Failed to save cache: ", err)
	}

	pterm.Success.Printf("Built %s in %s\n", opts.Name, res.Duration.Round(time.Millisecond))
	Notify("Build Success", opts.Name, false)
	Stats.Add(res)
	return nil
}

// runGoBuildWithProgress запускает go build и рисует UI
func runGoBuildWithProgress(args []string) error {
	cmd := exec.Command("go", args...)
	cmd.Env = os.Environ()

	// Перехватываем потоки
	stderr, _ := cmd.StderrPipe()
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		return err
	}

	spinner, _ := pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		WithText("Initializing build...").
		Start()

	// Используем WaitGroup, чтобы дождаться завершения обработки логов
	var wg sync.WaitGroup
	var compileErr error
	var compileOutput strings.Builder

	// 1. Обработка STDOUT (JSON от go build)
	wg.Add(1)
	go func() {
		defer wg.Done()

		scanner := bufio.NewScanner(stdout)
		// Увеличим буфер, на случай длинных строк JSON
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		type GoEvent struct {
			Action     string `json:"Action"`
			ImportPath string `json:"ImportPath"`
			Output     string `json:"Output"`
		}

		for scanner.Scan() {
			line := scanner.Bytes()

			// Пропускаем пустые строки или не JSON (бывает при смешанном выводе)
			if len(line) == 0 || line[0] != '{' {
				continue
			}

			var event GoEvent
			if err := json.Unmarshal(line, &event); err != nil {
				// Если JSON битый, сохраняем ошибку, но пробуем читать дальше
				compileErr = fmt.Errorf("failed to parse json output: %w", err)
				continue
			}

			// Обновляем UI
			if event.Action == "build-output" {
				pkgDisplay := event.ImportPath
				if len(pkgDisplay) > 50 {
					parts := strings.Split(pkgDisplay, "/")
					if len(parts) > 2 {
						pkgDisplay = ".../" + strings.Join(parts[len(parts)-2:], "/")
					}
				}
				spinner.UpdateText(fmt.Sprintf("Compiling: %s", pkgDisplay))
			}

			// Если есть вывод (ошибки компилятора)
			if event.Output != "" {
				compileOutput.WriteString(event.Output)
			}
		}

		// Проверяем ошибки самого сканера
		if err := scanner.Err(); err != nil {
			compileErr = fmt.Errorf("error reading build output: %w", err)
		}
	}()

	// 2. Обработка STDERR (системные ошибки go)
	// Обычно stderr пуст при -json, если только не случилась паника или фатальная ошибка запуска
	go func() {
		scannerErr := bufio.NewScanner(stderr)
		for scannerErr.Scan() {
			spinner.Warning(scannerErr.Text())
		}
	}()

	// Ждем завершения процесса
	cmdErr := cmd.Wait()

	// Ждем, пока дочитаются все логи (важно!)
	wg.Wait()

	finalOutput := strings.TrimSpace(compileOutput.String())
	if finalOutput != "" {
		err := spinner.Stop()
		if err != nil {
			pterm.Error.Printfln("Failed to stop ui spinner: %s", err)
			return err
		}
		pterm.Info.Println("Compiler output:")
		outputLines := strings.Split(strings.TrimSpace(finalOutput), "\n")
		for _, line := range outputLines {
			line = filepath.ToSlash(line)
			fmt.Printf("  | %s\n", line)
		}
	}

	if cmdErr != nil {
		spinner.Fail("Build Failed\n")
		return cmdErr
	}

	// Если процесс завершился успешно, но были ошибки парсинга/чтения
	if compileErr != nil {
		spinner.Warning("Build finished, but output parsing failed")
		return compileErr
	}

	pterm.Success.Println("Compilation finished")
	return nil
}

// resolveGlobs превращает паттерны в список файлов
func resolveGlobs(patterns []string) ([]string, error) {
	var files []string
	for _, p := range patterns {
		matches, err := doublestar.FilepathGlob(p)
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	return files, nil
}
