package systems

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/logger"
	"github.com/sirupsen/logrus"
)

// Мультипликаторы для трансформации координат в 8 октантов
var multipliers = [4][8]int{
	{1, 0, 0, -1, -1, 0, 0, 1},
	{0, 1, -1, 0, 0, -1, 1, 0},
	{0, 1, 1, 0, 0, -1, -1, 0},
	{1, 0, 0, 1, -1, 0, 0, -1},
}

// ComputeVisibleTiles возвращает мапу индексов {index: true}, которые видны.
func ComputeVisibleTiles(w *domain.GameWorld, pos domain.Position, vision *domain.VisionComponent) map[int]bool {
	fovLogger := logger.Log.WithFields(logrus.Fields{
		"component":    "fov_system",
		"observer_pos": pos,
	})

	// 1. Если Всевидящий (ГМ, Птица) -> возвращаем nil (маркер "Вижу всё")
	if vision != nil && vision.Omniscient {
		fovLogger.WithField("is_omniscient", true).Debug("FOV calculation skipped for omniscient observer.")
		return nil // Signal: Full Visibility
	}

	radius := 8 // Значение по умолчанию
	if vision != nil {
		radius = vision.Radius
	}

	fovLogger.WithField("radius", radius).Debug("Starting FOV calculation.")

	visibleMap := make(map[int]bool)
	if radius <= 0 {
		fovLogger.Warn("FOV calculation skipped for blind observer (radius <= 0).")
		return visibleMap // Слепой
	}

	// 2. Центр всегда виден
	visibleMap[w.GetIndex(pos.X, pos.Y)] = true

	// 3. Запускаем рекурсивный Shadowcasting для 8 октантов
	for i := 0; i < 8; i++ {
		castLight(w, pos.X, pos.Y, 1, 1.0, 0.0, radius,
			multipliers[0][i], multipliers[1][i],
			multipliers[2][i], multipliers[3][i], visibleMap)
	}

	fovLogger.WithField("visible_tiles", len(visibleMap)).Debug("FOV calculation complete.")

	return visibleMap
}

func castLight(w *domain.GameWorld, cx, cy, row int, start, end float64, radius, xx, xy, yx, yy int, visibleMap map[int]bool) {
	if start < end {
		return
	}

	radiusSq := float64(radius * radius)

	for j := row; j <= radius; j++ {
		dx, dy := -j-1, -j
		blocked := false
		newStart := start

		for {
			dx++
			if dx > 0 {
				break
			}
			dy = -j

			// Расчет наклонов (Slopes)
			lSlope := (float64(dx) - 0.5) / (float64(dy) + 0.5)
			rSlope := (float64(dx) + 0.5) / (float64(dy) - 0.5)

			if start < rSlope {
				continue
			}
			if end > lSlope {
				break
			}

			// Трансформация координат в глобальные
			X := cx + dx*xx + dy*xy
			Y := cy + dx*yx + dy*yy

			// Проверка границ и радиуса
			if X >= 0 && Y >= 0 && X < w.Width && Y < w.Height {
				if float64(dx*dx+dy*dy) < radiusSq {
					idx := w.GetIndex(X, Y)
					visibleMap[idx] = true
				}
			}

			// Логика теней
			if blocked {
				// Мы идем вдоль стены...
				if isBlocking(w, X, Y) {
					newStart = rSlope
					continue
				} else {
					// Стена кончилась, началась пустота
					blocked = false
					start = newStart
				}
			} else {
				// Мы шли по пустоте и наткнулись на стену
				if isBlocking(w, X, Y) && j < radius {
					blocked = true
					// Рекурсивно запускаем сканирование следующего ряда
					castLight(w, cx, cy, j+1, start, lSlope, radius, xx, xy, yx, yy, visibleMap)
					newStart = rSlope
				}
			}
		}
		if blocked {
			break
		}
	}
}

// isBlocking проверяет, блокирует ли клетка взгляд
func isBlocking(w *domain.GameWorld, x, y int) bool {
	// Выход за границы считается блокирующим
	if x < 0 || y < 0 || x >= w.Width || y >= w.Height {
		return true
	}
	// Проверяем стену
	// В будущем здесь можно проверять компонент Physics.IsTransparent
	return w.Map[y][x].IsWall
}
