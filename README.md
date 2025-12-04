# Cognitive Dungeon: Backend (Go)

Серверная часть проекта **Cognitive Dungeon** — текстовой онлайн RPG с гибридным ИИ.
Написана на **Go**, использует **WebSockets** для общения с клиентом.

## 🛠 Технологии
- **Language:** Go (Golang) 1.22+
- **Protocol:** JSON over WebSockets
- **Libraries:** `gorilla/websocket`
- **Architecture:** Client-Server authoritative (Server handles logic)

## 🚀 Запуск локально

1. **Установите Go:** [Скачать Go](https://go.dev/dl/)
2. **Клонируйте репозиторий:**
   ```bash
   git clone https://github.com/YOUR_USER/cd-backend-go.git
   cd cd-backend-go
   ```
3. **Установите зависимости:**
   ```bash
   go mod tidy
   ```
4. **Запустите сервер:**
   ```bash
   go run cmd/server/main.go
   ```
   *Вы увидите сообщение: `🛡️ Cognitive Dungeon Server running on :8080`*

## 🔌 API (WebSocket)

**Адрес:** `ws://localhost:8080/ws`

### Формат сообщений (JSON)

**Клиент -> Сервер (Command):**
```json
{
  "action": "MOVE",
  "payload": {
    "dx": 1, 
    "dy": 0
  }
}
```

**Сервер -> Клиент (Response):**
```json
{
  "type": "UPDATE",
  "player": { ... },
  "logs": [ { "text": "Вы сделали шаг", "type": "INFO" } ]
}
```

## 📂 Структура проекта
- `cmd/server/` — Точка входа (`main.go`).
- `internal/engine/` — Игровая логика (Game Loop, Physics).
- `internal/models/` — Структуры данных (JSON Contract).
- `pkg/` — Утилиты и генераторы.
