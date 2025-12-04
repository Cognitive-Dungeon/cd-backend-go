package engine

import (
	"cognitive-server/internal/models"
	"encoding/json"
	"testing"
)

// Helper: Создает мини-мир 3x3 для тестов
// [ . . . ] (0,0) (1,0) (2,0)
// [ . # . ] (0,1) (1,1) (2,1)  <- Стена в центре
// [ . . . ] (0,2) (1,2) (2,2)
func createTestEngine() *GameEngine {
	world := &models.GameWorld{
		Width: 3, Height: 3,
		Map: make([][]models.Tile, 3),
	}

	for y := 0; y < 3; y++ {
		row := make([]models.Tile, 3)
		for x := 0; x < 3; x++ {
			row[x] = models.Tile{X: x, Y: y, IsWall: false, Env: "floor"}
		}
		world.Map[y] = row
	}

	// Ставим стену в центре (1,1)
	world.Map[1][1].IsWall = true

	player := &models.Entity{
		ID: "p1", Pos: models.Position{X: 0, Y: 0},
		Stats: models.Stats{HP: 100},
	}

	return &GameEngine{
		World:  world,
		Player: player,
		Logs:   []models.LogEntry{},
	}
}

func TestMove_Success(t *testing.T) {
	g := createTestEngine()
	// Игрок в 0,0. Идем вправо (1,0)

	payload, _ := json.Marshal(models.MovePayload{Dx: 1, Dy: 0})
	cmd := models.ClientCommand{Action: "MOVE", Payload: payload}

	resp := g.ProcessCommand(cmd)

	if g.Player.Pos.X != 1 || g.Player.Pos.Y != 0 {
		t.Errorf("Expected pos (1,0), got (%d,%d)", g.Player.Pos.X, g.Player.Pos.Y)
	}

	// Проверяем, что в ответе есть мир
	if resp.World == nil {
		t.Error("Response should contain updated World state")
	}
}

func TestMove_Collision(t *testing.T) {
	g := createTestEngine()
	// Ставим игрока в (0,1), справа от него стена в (1,1)
	g.Player.Pos = models.Position{X: 0, Y: 1}

	// Пытаемся пойти в стену (вправо)
	payload, _ := json.Marshal(models.MovePayload{Dx: 1, Dy: 0})
	cmd := models.ClientCommand{Action: "MOVE", Payload: payload}

	g.ProcessCommand(cmd)

	// Координаты не должны измениться
	if g.Player.Pos.X != 0 || g.Player.Pos.Y != 1 {
		t.Error("Player moved into a wall!")
	}

	// Проверяем лог
	if len(g.Logs) == 0 {
		t.Error("Expected error log message")
	}
}

func TestMove_OutOfBounds(t *testing.T) {
	g := createTestEngine()
	// Игрок в 0,0. Идем влево (-1, 0) - за пределы карты

	payload, _ := json.Marshal(models.MovePayload{Dx: -1, Dy: 0})
	cmd := models.ClientCommand{Action: "MOVE", Payload: payload}

	g.ProcessCommand(cmd)

	if g.Player.Pos.X != 0 {
		t.Error("Player moved out of bounds!")
	}
}
