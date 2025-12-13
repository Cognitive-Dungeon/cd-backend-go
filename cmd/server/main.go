package main

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine"
	"cognitive-server/internal/version"
	"cognitive-server/pkg/api" // Нужно для шаблонов при спавне
	"cognitive-server/pkg/dungeon"
	"cognitive-server/pkg/logger"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Конфигурация тайм-аутов
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Инициализируем игровой сервис (Арбитр).
// NewService() теперь создает все уровни и сущности.
var gameInstance *engine.GameService

// Client - посредник между Websocket и GameService
type Client struct {
	game     *engine.GameService
	conn     *websocket.Conn
	send     chan api.ServerResponse
	entityID string
}

// readPump читает команды от клиента
func (c *Client) readPump() {
	defer func() {
		c.game.Hub.Unregister(c.entityID)
		if err := c.conn.Close(); err != nil {
			logger.Log.WithError(err).Warn("failed to close websocket connection")
		}
		// Освобождаем сущность, чтобы AI мог перехватить управление (если захотим)
		// или просто чтобы пометить, что игрок оффлайн
		if ent := c.game.GetEntity(c.entityID); ent != nil {
			ent.ControllerID = ""
			logger.Log.WithField("entity_id", c.entityID).Info("Client disconnected")
			// Сообщаем движку, что игрок ушел, чтобы прервать его ход немедленно
			// Используем select, чтобы не заблокировать readPump, если канал полон (маловероятно, но безопасно)
			select {
			case c.game.DisconnectChan <- c.entityID:
			default:
			}
		}
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		logger.Log.WithError(err).Warn("failed to set read deadline")
	}
	c.conn.SetPongHandler(func(string) error {
		if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
			logger.Log.WithError(err).Warn("failed to set pong read deadline")
		}
		return nil
	})

	// 1. HANDSHAKE (LOGIN)
	var loginCmd api.ClientCommand
	if err := c.conn.ReadJSON(&loginCmd); err != nil {
		logger.Log.Warn("Handshake failed")
		return
	}

	c.entityID = loginCmd.Token
	if c.entityID == "" {
		c.entityID = domain.GenerateID()
	}

	// 2. ПОИСК ИЛИ СОЗДАНИЕ ИГРОКА
	ent := c.game.GetEntity(c.entityID)
	if ent == nil {
		logger.Log.Infof("Player %s not found. Spawning...", c.entityID)
		newPlayer := dungeon.CreatePlayer(c.entityID)

		// Ищем место для спавна на уровне 0
		world := c.game.Worlds[0]
		placed := false
		// Сканируем центр карты
		for y := 10; y < 20; y++ {
			for x := 15; x < 25; x++ {
				if !world.Map[y][x].IsWall && len(world.GetEntitiesAt(x, y)) == 0 {
					newPlayer.Pos = domain.Position{X: x, Y: y}
					placed = true
					goto Done
				}
			}
		}
	Done:
		if !placed {
			newPlayer.Pos = domain.Position{X: 1, Y: 1} // Fallback
		}

		// Отправляем в движок через канал
		c.game.JoinChan <- newPlayer

		// Даем движку мгновение на обработку
		time.Sleep(50 * time.Millisecond)
		ent = newPlayer
	}

	ent.ControllerID = "session_" + c.entityID
	logger.Log.WithFields(logrus.Fields{
		"entity_id": c.entityID,
		"name":      ent.Name,
	}).Info("Client logged in")

	// 3. ПОДПИСКА НА ОБНОВЛЕНИЯ
	gameUpdates := c.game.Hub.Register(c.entityID)

	// Запускаем пересылку обновлений из Hub в writePump
	go func() {
		for msg := range gameUpdates {
			c.send <- msg
		}
		close(c.send)
	}()

	// Отправляем INIT (триггер первой отрисовки)
	c.game.ProcessCommand(api.ClientCommand{Action: "INIT", Token: c.entityID})

	// 4. ЦИКЛ ЧТЕНИЯ КОМАНД
	for {
		var cmd api.ClientCommand
		err := c.conn.ReadJSON(&cmd)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Log.Errorf("WS Error: %v", err)
			}
			break
		}
		cmd.Token = c.entityID
		c.game.ProcessCommand(cmd)
	}
}

// writePump отправляет данные клиенту + Ping
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if err := c.conn.Close(); err != nil {
			logger.Log.WithError(err).Warn("failed to close websocket connection in writePump")
		}
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Log.WithError(err).Warn("failed to set write deadline")
			}
			if !ok {
				if err := c.conn.WriteMessage(websocket.CloseMessage, []byte{}); err != nil {
					logger.Log.WithError(err).Debug("write close message failed")
				}
				return
			}
			if err := c.conn.WriteJSON(message); err != nil {
				logger.Log.WithError(err).Debug("write json message failed")
				return
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Log.WithError(err).Warn("failed to set ping write deadline")
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Log.WithError(err).Debug("ping failed")
				return
			}
		}
	}
}

func serveWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Log.Error("Upgrade error:", err)
		return
	}

	client := &Client{
		game: gameInstance,
		conn: conn,
		send: make(chan api.ServerResponse, 256),
	}

	// Запускаем две горутины на каждое соединение
	go client.writePump()
	go client.readPump()
}

func serveVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(version.Info())
}

func serveHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func init() {
	logger.Init()
}

func main() {
	logger.Log.Info("Starting Cognitive Dungeon...")
	logger.Log.Info(version.String())
	port := os.Getenv("CD_PORT")
	if port == "" {
		port = "8080"
	}

	// --- Порядок запуска ---
	// 0. Инициализируем сервис (после того как логгер уже готов)
	gameInstance = engine.NewService()

	// 1. Игровой сервис инициализирован выше (var gameInstance).
	//    На этом этапе все миры, NPC и предметы уже созданы в памяти.

	// 2. Запускаем игровой цикл в фоновой горутине.
	//    Мир начинает "жить" своей жизнью (ALife симуляция).
	logger.Log.Info("Starting Game Loop...")
	gameInstance.Start()

	http.HandleFunc("/ws", serveWs)
	http.HandleFunc("/version", serveVersion)
	http.HandleFunc("/health", serveHealth)

	logger.Log.Infof("🛡️  Cognitive Dungeon Server running on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		logger.Log.Fatal("ListenAndServe error:", err)
	}
}
