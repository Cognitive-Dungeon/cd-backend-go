package dungeon

import (
	"cognitive-server/internal/domain"
	"math/rand"
	"time"
)

// Константы генерации
const (
	MapWidth  = 40
	MapHeight = 25
	MaxRooms  = 8
	MinSize   = 4
	MaxSize   = 10
)

// Generate создает новый уровень, используя LevelBuilder.
// Теперь это высокоуровневая функция-директор, определяющая "рецепт" уровня.
func Generate(level int, r *rand.Rand) (*domain.GameWorld, []domain.Entity, domain.Position) {
	// Если рандом не передан, создаем свой (хотя лучше передавать извне)
	if r == nil {
		r = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	// 1. Инициализируем билдер
	builder := NewLevel(level, r).
		WithSize(MapWidth, MapHeight).
		WithRooms(MaxRooms)

	// 2. Размещаем выходы
	// Логика внутри PlaceExit сама свяжет ID выходов
	builder.PlaceExit("up", level-1)
	builder.PlaceExit("down", level+1)

	// 3. Настраиваем сложность (Количество врагов)
	// Чем глубже, тем больше врагов
	baseMobCount := 3 + (level / 2)

	// Спавним Гоблинов (есть всегда)
	builder.SpawnEnemy("goblin", baseMobCount)

	// Спавним Орков (начиная со 2-го уровня)
	if level >= 2 {
		orcCount := 1 + (level / 3)
		builder.SpawnEnemy("orc", orcCount)
	}

	// Спавним Троллей (начиная с 5-го уровня, редко)
	if level >= 5 && r.Float32() > 0.7 {
		builder.SpawnEnemy("troll", 1)
	}

	// 4. Разбрасываем лут
	// От 3 до 6 предметов на уровень
	itemCount := 3 + r.Intn(4)

	// Если таблица лута пуста (например, шаблоны не загрузились), пропускаем
	if len(LootTable) > 0 {
		for i := 0; i < itemCount; i++ {
			// Берем случайный ключ из глобальной таблицы
			randomItemKey := LootTable[r.Intn(len(LootTable))]
			builder.SpawnItem(randomItemKey, 1)
		}
	}

	// 5. Собираем уровень
	return builder.Build()
}
