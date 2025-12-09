package main

import (
	"cognitive-server/internal/engine"
	"cognitive-server/pkg/api"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket" // Исправлена возможная опечатка comcom -> com
)

var upgrader = websocket.Upgrader{
	// Разрешаем CORS запросы (нужно для разработки React на другом порту)
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Инициализируем игровой сервис (Арбитр).
// NewService() теперь создает все уровни и сущности.
var gameInstance = engine.NewService()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Failed to upgrade connection:", err)
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
	// --- ИЗМЕНЕНИЕ: Используем новый метод для поиска сущности ---
	// Теперь нам не нужно знать, на каком уровне находится сущность при логине.
	ent := gameInstance.GetEntity(entityID)
	// -----------------------------------------------------------
	if ent == nil {
		log.Printf("Login failed: unknown entity '%s'", entityID)
		conn.WriteJSON(map[string]string{"error": "Entity not found"})
		return
	}

	// Помечаем, что сущность управляется человеком
	ent.ControllerID = "session_" + entityID[:4]

	log.Printf("Client connected and possessed %s (%s) on level %d", ent.Name, entityID, ent.Level)

	// 2. Регистрация в Хабе для получения обновлений
	clientChan := gameInstance.Hub.Register(entityID)
	defer func() {
		gameInstance.Hub.Unregister(entityID)
		ent.ControllerID = "" // Освобождаем сущность при дисконнекте
		log.Printf("Client disconnected: %s", entityID)
	}()

	// 3. Отправляем начальное состояние мира
	// Команда INIT просто триггерит отправку ServerResponse без траты хода
	gameInstance.ProcessCommand(api.ClientCommand{Action: "INIT", Token: entityID})

	// 4. Запускаем горутины для чтения и записи сообщений
	// Writer: читает из канала и отправляет в WebSocket
	go func() {
		for event := range clientChan {
			if err := conn.WriteJSON(event); err != nil {
				// Канал был закрыт или соединение разорвано
				return
			}
		}
	}()

	// Reader: читает из WebSocket и отправляет в движок
	for {
		var cmd api.ClientCommand
		if err := conn.ReadJSON(&cmd); err != nil {
			// Соединение разорвано
			break
		}

		// ВАЖНО: Принудительно устанавливаем ID сущности из контекста соединения.
		// Это мера безопасности, чтобы клиент не мог управлять чужими персонажами.
		cmd.Token = entityID
		gameInstance.ProcessCommand(cmd)
	}
}

func main() {
	port := os.Getenv("CD_PORT")
	if port == "" {
		port = "8080"
	}

	// --- Порядок запуска ---
	// 1. Игровой сервис инициализирован выше (var gameInstance).
	//    На этом этапе все миры, NPC и предметы уже созданы в памяти.

	// 2. Запускаем игровой цикл в фоновой горутине.
	//    Мир начинает "жить" своей жизнью (ALife симуляция).
	log.Println("Starting Game Loop...")
	gameInstance.Start()

	// 3. Настраиваем обработчик для WebSocket-подключений.
	http.HandleFunc("/ws", wsHandler)

	// 4. Запускаем веб-сервер, который будет принимать подключения от игроков.
	log.Println("🛡️  Cognitive Dungeon Server running on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe error:", err)
	}
}
