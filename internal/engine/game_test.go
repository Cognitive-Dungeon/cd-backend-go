package engine

import (
	"cognitive-server/internal/models"
	"encoding/json"
	"testing"
)

func TestMovement(t *testing.T) {
	game := NewGame()
	startX := game.Player.Pos.X

	// Формируем команду движения вправо (+1 по X)
	payload, _ := json.Marshal(models.MovePayload{Dx: 1, Dy: 0})
	cmd := models.ClientCommand{Action: "MOVE", Payload: payload}

	// Выполняем
	game.ProcessCommand(cmd)

	// Проверяем
	if game.Player.Pos.X != startX+1 {
		t.Errorf("Expected X to be %d, got %d", startX+1, game.Player.Pos.X)
	}
}

func TestWallCollision(t *testing.T) {
	game := NewGame()
	// Поставим игрока в 1,1
	game.Player.Pos.X = 1
	game.Player.Pos.Y = 1
	// Поставим стену в 2,1
	game.World.Map[1][2].IsWall = true

	// Пытаемся пойти в стену
	payload, _ := json.Marshal(models.MovePayload{Dx: 1, Dy: 0})
	game.ProcessCommand(models.ClientCommand{Action: "MOVE", Payload: payload})

	// Он не должен был сдвинуться
	if game.Player.Pos.X != 1 {
		t.Error("Player walked through wall!")
	}
}
