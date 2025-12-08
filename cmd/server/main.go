package main

import (
	"cognitive-server/internal/engine"
	"cognitive-server/pkg/api"
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
var gameInstance = engine.NewService()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 1. HANDSHAKE / LOGIN
	// Читаем первое сообщение, ожидаем { "action": "LOGIN", "token": "entity_uuid" }
	var loginCmd api.ClientCommand
	if err := conn.ReadJSON(&loginCmd); err != nil {
		log.Println("Handshake error:", err)
		return
	}

	entityID := loginCmd.Token
	// Валидация: существует ли такая сущность?
	ent := gameInstance.World.GetEntity(entityID)
	if ent == nil {
		log.Println("Login failed: unknown entity", entityID)
		conn.WriteJSON(map[string]string{"error": "Entity not found"})
		return
	}

	// Помечаем, что сущность управляется (опционально, для логики пропуска хода AI)
	ent.ControllerID = "session_" + entityID[:4]

	log.Printf("Client connected as %s (%s)", ent.Name, entityID)

	// 2. Регистрация в Хабе
	clientChan := gameInstance.Hub.Register(entityID)
	defer func() {
		gameInstance.Hub.Unregister(entityID)
		ent.ControllerID = "" // Освобождаем сущность при дисконнекте
		log.Printf("Client disconnected: %s", entityID)
	}()

	// 3. Отправляем начальное состояние
	gameInstance.ProcessCommand(api.ClientCommand{Action: "INIT", Token: entityID})

	// 4. Каналы (Write/Read)
	// Writer
	go func() {
		for event := range clientChan {
			if err := conn.WriteJSON(event); err != nil {
				return
			}
		}
	}()

	// Reader
	for {
		var cmd api.ClientCommand
		if err := conn.ReadJSON(&cmd); err != nil {
			break
		}

		// ВАЖНО: Форсируем Token из контекста соединения (безопасность)
		// Чтобы клиент не мог прислать action MOVE с чужим token
		cmd.Token = entityID
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
