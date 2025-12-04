package engine

import (
	"cognitive-server/internal/models"
	"testing"
)

// Helper: Создает игру с Игроком и одним Врагом
func setupLoopTest() (*GameEngine, *models.Entity) {
	g := NewGame()

	// Очищаем карту от мусора генератора
	g.Entities = []models.Entity{}
	g.World.Map = make([][]models.Tile, 10)
	for y := 0; y < 10; y++ {
		g.World.Map[y] = make([]models.Tile, 10) // 10x10 пустой пол
	}
	g.World.Width = 10
	g.World.Height = 10

	// Игрок в 0,0
	g.Player.Pos = models.Position{X: 0, Y: 0}
	g.Player.NextActionTick = 0
	g.Player.Stats.HP = 100

	// Враг в 0,1 (рядом)
	enemy := models.Entity{
		ID: "goblin_1", Name: "Гоблин", IsHostile: true,
		Pos:            models.Position{X: 0, Y: 1},
		Stats:          models.Stats{HP: 20, Strength: 5},
		NextActionTick: 0,
	}
	g.Entities = append(g.Entities, enemy)

	return g, &g.Entities[0] // Возвращаем ссылку на движок и врага
}

func TestGameLoop_Wait(t *testing.T) {
	g, _ := setupLoopTest()
	initialTick := g.World.GlobalTick // 0

	// Игрок ждет (Action Cost = 50)
	cmd := models.ClientCommand{Action: "WAIT"}
	g.ProcessCommand(cmd)

	// 1. Проверяем, что время игрока ушло вперед
	if g.Player.NextActionTick != initialTick+models.TimeCostWait {
		t.Errorf("Player tick mismatch. Got %d, want %d", g.Player.NextActionTick, initialTick+models.TimeCostWait)
	}

	// 2. Проверяем, что глобальное время догнало игрока
	// (Так как NPC тоже сходил и его время тоже увеличилось)
	if g.World.GlobalTick < g.Player.NextActionTick {
		t.Errorf("Global tick did not catch up. Got %d", g.World.GlobalTick)
	}
}

func TestGameLoop_NPC_Attack(t *testing.T) {
	g, _ := setupLoopTest()
	initialHP := g.Player.Stats.HP

	// Ситуация: Враг стоит в (0,1), Игрок в (0,0).
	// Оба на тике 0.
	// Игрок делает шаг ВЛЕВО (в стену/никуда) или ЖДЕТ.
	// Это триггерит GameLoop.
	// Враг должен ударить, так как он рядом.

	cmd := models.ClientCommand{Action: "WAIT"}
	g.ProcessCommand(cmd)

	// Проверяем, что HP игрока уменьшилось (Гоблин ударил)
	if g.Player.Stats.HP >= initialHP {
		t.Error("NPC failed to attack player during the loop")
	}

	// Проверяем логи
	foundLog := false
	for _, l := range g.Logs {
		if l.Type == "COMBAT" {
			foundLog = true
			break
		}
	}
	if !foundLog {
		t.Error("Expected COMBAT log not found")
	}
}

func TestGameLoop_TurnOrder(t *testing.T) {
	g, enemy := setupLoopTest()

	// Сценарий: Игрок очень медленный (уже потратил время), Враг быстрый.
	g.Player.NextActionTick = 200
	enemy.NextActionTick = 100 // Враг должен сходить СЕЙЧАС (до игрока)

	// Запускаем луп вручную, как будто игрок только что закончил действие
	g.RunGameLoop()

	// Враг должен был сходить.
	// Если он ударил (Light Attack = 80), его тик станет 100 + 80 = 180.
	// 180 < 200, значит он должен сходить ЕЩЕ РАЗ!
	// 180 + 80 = 260.
	// 260 > 200. Теперь очередь Игрока. Цикл должен остановиться.

	if enemy.NextActionTick <= 200 {
		t.Errorf("Enemy should have acted multiple times until passing player tick (200). Current: %d", enemy.NextActionTick)
	}

	// Глобальное время должно остановиться на времени игрока (200),
	// так как сейчас ЕГО очередь.
	if g.World.GlobalTick != 200 {
		t.Errorf("Global tick should match player tick when giving control back. Got %d", g.World.GlobalTick)
	}
}
