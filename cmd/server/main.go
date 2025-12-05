package main

import (
	"cognitive-server/internal/core"
	"cognitive-server/internal/domain"
	"log"
	"net/http"
	"os"
	"time" // Добавили для time.Sleep

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // CORS
}

// Создаем инстанс, но пока не запускаем
var gameInstance = core.NewService()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("Player connected")

	// --- 1. INIT ---
	// Отправляем команду инициализации в движок
	gameInstance.ProcessCommand(domain.ClientCommand{Action: "INIT"})

	// ХАК: Даем движку 10мс прожевать команду (так как каналы асинхронны)
	// В будущем здесь будет ожидание события из канала обновлений
	time.Sleep(10 * time.Millisecond)

	// Берем текущее состояние вручную
	initResp := gameInstance.GetState()
	initResp.Type = "INIT" // Явно ставим тип для фронтенда

	if err := conn.WriteJSON(initResp); err != nil {
		log.Println("Write init error:", err)
		return
	}

	// --- 2. GAME LOOP (Слушаем сокет) ---
	for {
		var cmd domain.ClientCommand
		err := conn.ReadJSON(&cmd)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		log.Printf("Command received: %s\n", cmd.Action)

		// 1. Кидаем команду в канал движка (неблокирующая операция)
		gameInstance.ProcessCommand(cmd)

		// 2. ХАК: Ждем обработки (временное решение для совместимости с React)
		time.Sleep(10 * time.Millisecond)

		// 3. Забираем актуальное состояние мира
		resp := gameInstance.GetState()

		// 4. Отправляем клиенту
		err = conn.WriteJSON(resp)
		if err != nil {
			log.Println("Write error:", err)
			break
		}
	}
}

func main() {
	port := os.Getenv("CD_PORT")
	if port == "" {
		port = "8080"
	}

	// ВАЖНО: Запускаем игровой цикл в фоне перед стартом сервера
	log.Println("Starting Game Loop...")
	gameInstance.Start()

	http.HandleFunc("/ws", wsHandler)

	log.Println("🛡️  Cognitive Dungeon Server running on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
