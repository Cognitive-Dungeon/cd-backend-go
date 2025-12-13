package server

import (
	"cognitive-server/internal/engine"
	"encoding/json"
	"fmt"
	"net/http"
)

// DebugHandler предоставляет доступ к внутреннему состоянию движка
type DebugHandler struct {
	Service *engine.GameService
}

func NewDebugHandler(s *engine.GameService) *DebugHandler {
	return &DebugHandler{Service: s}
}

// RegisterRoutes регистрирует debug-эндпоинты
func (h *DebugHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/debug/worlds", h.handleListWorlds)
	mux.HandleFunc("/debug/entities", h.handleDumpEntities)
	mux.HandleFunc("/debug/queue", h.handleTurnQueue)
}

// /debug/worlds - список активных миров и количество сущностей в них
func (h *DebugHandler) handleListWorlds(w http.ResponseWriter, r *http.Request) {
	type WorldSummary struct {
		LevelID     int  `json:"level_id"`
		Width       int  `json:"width"`
		Height      int  `json:"height"`
		EntityCount int  `json:"entity_count"`
		IsActive    bool `json:"is_active"` // Добавим флаг для ясности
	}

	var summary []WorldSummary

	// Итерируемся по INSTANCES, так как именно они содержат актуальное состояние игры.
	// Динамически созданные уровни живут здесь.
	for id, instance := range h.Service.Instances {
		summary = append(summary, WorldSummary{
			LevelID:     id,
			Width:       instance.World.Width,
			Height:      instance.World.Height,
			EntityCount: len(instance.Entities),
			IsActive:    true,
		})
	}

	writeJSON(w, summary)
}

// /debug/entities?level=1 - дамп всех сущностей на уровне (включая скрытые параметры AI)
func (h *DebugHandler) handleDumpEntities(w http.ResponseWriter, r *http.Request) {
	levelStr := r.URL.Query().Get("level")
	var levelID int
	fmt.Sscanf(levelStr, "%d", &levelID)

	instance, ok := h.Service.Instances[levelID]
	if !ok {
		http.Error(w, "Instance not found or not active", http.StatusNotFound)
		return
	}

	// Мы возвращаем полные структуры domain.Entity, включая AI стейт, координаты и скрытые статы
	writeJSON(w, instance.Entities)
}

// /debug/queue?level=1 - просмотр очереди ходов
func (h *DebugHandler) handleTurnQueue(w http.ResponseWriter, r *http.Request) {
	levelStr := r.URL.Query().Get("level")
	var levelID int
	fmt.Sscanf(levelStr, "%d", &levelID)

	instance, ok := h.Service.Instances[levelID]
	if !ok {
		http.Error(w, "Instance not found", http.StatusNotFound)
		return
	}

	// Внимание: TurnQueue - это куча, порядок в слайсе может не соответствовать порядку извлечения,
	// но для дебага сойдет.
	type QueueItemView struct {
		EntityID string `json:"entity_id"`
		Name     string `json:"name"`
		Priority int    `json:"next_tick"`
	}

	// Временный доступ к очереди (требует аккуратности с concurrency, но для debug READ-only ок)
	// В идеале добавить метод GetQueueSnapshot() в TurnManager
	// Сейчас сделаем заглушку, так как TurnQueue приватная в пакете engine.

	dump := instance.TurnManager.DebugDump()
	writeJSON(w, dump)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	// Разрешаем запросы с любого источника (нужно для локального debug_client.html)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	w.Header().Set("Content-Type", "application/json")

	// Если data == nil (например, пустая очередь), возвращаем пустой массив [], а не null
	if data == nil {
		w.Write([]byte("[]"))
		return
	}

	json.NewEncoder(w).Encode(data)
}
