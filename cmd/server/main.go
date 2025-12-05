package main

import (
	"cognitive-server/internal/core"
	"cognitive-server/internal/domain"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // CORS
}

// В MVP один инстанс игры на всех
var gameInstance = core.NewService()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("Player connected")

	// 1. Отправляем INIT состояние
	initResp := gameInstance.ProcessCommand(domain.ClientCommand{Action: "INIT"})
	conn.WriteJSON(initResp)

	for {
		// 2. Читаем команду
		var cmd domain.ClientCommand
		err := conn.ReadJSON(&cmd)
		if err != nil {
			log.Println("Read error:", err)
			break
		}

		log.Printf("Command received: %s\n", cmd.Action)

		// 3. Обрабатываем
		resp := gameInstance.ProcessCommand(cmd)

		// 4. Отправляем ответ
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

	http.HandleFunc("/ws", wsHandler)

	log.Println("🛡️  Cognitive Dungeon Server running on :8080")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
