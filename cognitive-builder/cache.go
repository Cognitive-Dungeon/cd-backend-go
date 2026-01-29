package cognitive_builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
)

var cacheFile = filepath.Join(Dirs.Build, ".build_cache.json")

// CacheState хранит состояние сборки
type CacheState map[string]string // TargetName -> Hash

// loadCache загружает кэш с диска
func loadCache() CacheState {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return make(CacheState)
	}
	var state CacheState
	if err := json.Unmarshal(data, &state); err != nil {
		return make(CacheState)
	}
	return state
}

// saveCache сохраняет кэш
func (c CacheState) save() error {
	// Создаем папки если их нет
	if err := ensureDirs(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cacheFile, data, 0644)
}

// calculateSourcesHash считает общий хэш для набора файлов/папок
func calculateSourcesHash(patterns []string) (string, error) {
	var files []string

	// 1. Находим все файлы по паттернам
	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return "", err
		}
		files = append(files, matches...)
	}

	// 2. Сортируем для детерминизма
	sort.Strings(files)

	hasher := sha256.New()

	// 3. Хэшируем контент
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			return "", err
		}

		// Пишем имя файла в хэш (чтобы переименование влияло)
		hasher.Write([]byte(file))

		// Пишем контент
		if _, err := io.Copy(hasher, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// calculateHashForFiles считает хэш для конкретного списка файлов
func calculateHashForFiles(files []string) (string, error) {
	// 1. Сортируем для детерминизма
	sort.Strings(files)

	// Удаляем дубликаты (они могут появиться, если go list и Globs пересеклись)
	files = uniqueStrings(files)

	hasher := sha256.New()

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}

		f, err := os.Open(file)
		if err != nil {
			return "", err
		}

		// Пишем имя файла (относительно корня)
		// Чтобы переименование файла меняло хэш
		hasher.Write([]byte(file))

		// Пишем контент
		if _, err := io.Copy(hasher, f); err != nil {
			f.Close()
			return "", err
		}
		f.Close()
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func uniqueStrings(input []string) []string {
	keys := make(map[string]bool)
	var list []string
	for _, entry := range input {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
