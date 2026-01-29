package main

import (
	"path/filepath"

	cb "cognitive-builder"

	"github.com/magefile/mage/mg"
)

type Assets mg.Namespace

// Tiles генерирует атласы тайлов и определения для Tiled
func (Assets) Tiles() error {
	// Пути (можно вынести в константы)
	inputConfig := filepath.Join(cb.Dirs.AssetsRaw, "tiles_def.jsonc")
	fontPath := filepath.Join("cognitive-tools", "tileset_gen", "unifont-17.0.03.ttf")

	serverOut := cb.Dirs.AssetsOut
	tiledOut := filepath.Join("cognitive-tools", "tiled", "assets")

	return cb.Tool("tileset_gen").
		// Где лежат исходники тулзы (относительно корня репо)
		FromSource("cognitive-tools/tileset_gen").

		// От чего зависит пересборка?
		Input(inputConfig).
		Input(fontPath).

		// Что ожидаем на выходе? (если удалим эти файлы, сборка запустится снова)
		Output(filepath.Join(serverOut, "materials.json")). // Пример

		// Аргументы запуска
		Arg("-in", inputConfig).
		Arg("-font", fontPath).
		Arg("-server-out-dir", serverOut).
		Arg("-tiled-out-dir", tiledOut).
		Run()
}

// CookAll собирает все ассеты
func (Assets) CookAll() {
	mg.SerialDeps(Assets.Tiles)
}
