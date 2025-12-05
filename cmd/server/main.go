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
	// Разрешаем CORS запросы (нужно для разработки React на другом порту)
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Инициализируем игровой сервис (Арбитр)
var gameInstance = core.NewService()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	log.Println("Client connected")

	// 1. Подписка
	clientChan := gameInstance.Hub.Subscribe()
	defer gameInstance.Hub.Unsubscribe(clientChan)

	// 2. Инициализация
	// Это сообщение уйдет в движок, он сгенерирует ответ и
	// пришлет его обратно в clientChan через broadcast.
	gameInstance.ProcessCommand(domain.ClientCommand{Action: "INIT"})

	// 3. Запуск писателя (Server -> Client)
	go func() {
		for event := range clientChan {
			if err := conn.WriteJSON(event); err != nil {
				log.Println("Write error:", err)
				return
			}
		}
	}()

	// 4. Запуск читателя (Client -> Server)
	for {
		var cmd domain.ClientCommand
		err := conn.ReadJSON(&cmd)
		if err != nil {
			log.Println("Read error / Disconnect:", err)
			break
		}

		log.Printf("Command received: %s\n", cmd.Action)
		gameInstance.ProcessCommand(cmd)
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
