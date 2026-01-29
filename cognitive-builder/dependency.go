package cognitive_builder

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GetTransitiveDeps возвращает список всех локальных файлов (.go),
// от которых реально зависит указанный main пакет.
func GetTransitiveDeps(mainPkgPath string) ([]string, error) {
	// 1. Спрашиваем у Go список зависимостей
	// -deps: показать все транзитивные зависимости
	// -f {{.Dir}}: вывести только директории, где лежат эти пакеты
	cmd := exec.Command("go", "list", "-deps", "-f", "{{.Dir}}", "./"+mainPkgPath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	var files []string
	seenDirs := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		dir := strings.TrimSpace(scanner.Text())
		if dir == "" {
			continue
		}

		// 3. Фильтрация: Нас интересуют только папки внутри нашего проекта.
		rel, err := filepath.Rel(rootCwd(), dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			// Это внешняя зависимость (GOPATH или GOROOT), пропускаем
			continue
		}

		if seenDirs[dir] {
			continue
		}
		seenDirs[dir] = true

		// Собираем все .go файлы в этой директории
		matches, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		files = append(files, matches...)
	}

	return files, nil
}

// rootCwd запоминает, где мы запустили Mage
var _rootCwd string

func rootCwd() string {
	if _rootCwd == "" {
		_rootCwd, _ = os.Getwd()
	}
	return _rootCwd
}

func getModuleName() (string, error) {
	out, err := exec.Command("go", "list", "-m").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
