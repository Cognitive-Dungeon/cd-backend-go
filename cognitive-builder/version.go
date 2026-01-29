package cognitive_builder

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/magefile/mage/sh"
)

var buildEpoch = time.Date(2025, time.December, 4, 0, 0, 0, 0, time.UTC)

type VersionInfo struct {
	Tag       string // v0.1.0
	BuildNum  int    // Days since epoch
	Commit    string // a1b2c3d
	Branch    string // main
	Dirty     bool   // true если есть незакоммиченные изменения
	BuildTime string // RFC3339
}

func getVersion() VersionInfo {
	v := VersionInfo{
		BuildTime: time.Now().UTC().Format(time.RFC3339),
	}

	// 1. Считаем Build Number (дни с эпохи)
	// Делим часы на 24, отбрасываем дробную часть
	duration := time.Since(buildEpoch)
	v.BuildNum = int(duration.Hours() / 24)
	if v.BuildNum < 0 {
		v.BuildNum = 0
	}

	// 2. Получаем базовый тег (версию) из git
	// Если тегов нет, вернем v0.0.0
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	tag, err := cmd.Output()
	if err != nil {
		v.Tag = "v0.0.0"
	} else {
		v.Tag = strings.TrimSpace(string(tag))
	}

	// 3. Git Info
	v.Commit = gitInfo("rev-parse", "--short", "HEAD")
	v.Branch = gitInfo("branch", "--show-current")

	// 4. Dirty Check
	// Проверяем статус (porcelain дает чистый вывод для скриптов)
	status, _ := sh.Output("git", "status", "--porcelain")
	if strings.TrimSpace(status) != "" {
		v.Dirty = true
	}

	return v
}

// Генерируем LDFlags для вшивания в main/pkg/version
func (v VersionInfo) LDFlags(module string) string {
	pkg := fmt.Sprintf("%s/pkg/version", module)

	// Превращаем bool в строку для флага
	dirtyStr := "false"
	if v.Dirty {
		dirtyStr = "true"
	}

	flags := []string{
		fmt.Sprintf("-X '%s.Tag=%s'", pkg, v.Tag),
		fmt.Sprintf("-X '%s.BuildNum=%d'", pkg, v.BuildNum), // Вшиваем число!
		fmt.Sprintf("-X '%s.Commit=%s'", pkg, v.Commit),
		fmt.Sprintf("-X '%s.Branch=%s'", pkg, v.Branch),
		fmt.Sprintf("-X '%s.DirtyStr=%s'", pkg, dirtyStr), // Передаем как строку
		fmt.Sprintf("-X '%s.BuildTime=%s'", pkg, v.BuildTime),
		"-w", "-s",
	}
	return strings.Join(flags, " ")
}
