package handlers

import (
	"cognitive-server/internal/domain"
	"encoding/json"
)

// EntityFinder описывает любую структуру, которая может находить сущность по ID.
// GameService неявно реализует этот интерфейс.
type EntityFinder interface {
	GetEntity(id string) *domain.Entity
}

type WorldSwitcher interface {
	ChangeLevel(entity *domain.Entity, targetLevel int, targetPosID string)
}

// Context передает хендлеру состояние мира.
// Мы передаем ссылки, чтобы хендлер мог менять состояние (мутировать данные).
type Context struct {
	Finder   EntityFinder
	World    *domain.GameWorld
	Entities []*domain.Entity // Слайс сущностей
	Actor    *domain.Entity   // Тот, кто выполняет команду (Игрок или NPC)

	// --- Global Context for Events ---
	Worlds          map[int]*domain.GameWorld // Доступ ко всем мирам (нужен для переходов)
	AddGlobalEntity func(*domain.Entity)      // Коллбэк для регистрации новой сущности в глобальном стейте
	Switcher        WorldSwitcher
}

// Result - возвращает результат выполнения команды.
// Хендлер НЕ пишет в логи сервиса напрямую, он возвращает данные.
type Result struct {
	Msg     string          // Текст лога
	MsgType string          // Тип лога (INFO, COMBAT, SPEECH)
	Event   json.RawMessage // Сырые данные события для обработки движком
}

// HandlerFunc - это контракт для любой команды (MOVE, ATTACK, etc).
type HandlerFunc func(ctx Context, payload json.RawMessage) (Result, error)

// EmptyResult - вспомогательная функция для пустого успешного ответа
func EmptyResult() Result {
	return Result{}
}

// SpendActionPoints тратит время актора, если это применимо
func SpendActionPoints(actor *domain.Entity, cost int) {
	if actor.AI != nil {
		actor.AI.Wait(cost)
	}
}
