package handlers

import (
	"cognitive-server/internal/domain"
	"encoding/json"
)

// Context передает хендлеру состояние мира.
// Мы передаем ссылки, чтобы хендлер мог менять состояние (мутировать данные).
type Context struct {
	World    *domain.GameWorld
	Entities []*domain.Entity // Слайс сущностей
	Actor    *domain.Entity   // Тот, кто выполняет команду (Игрок или NPC)
}

// Result - возвращает результат выполнения команды.
// Хендлер НЕ пишет в логи сервиса напрямую, он возвращает данные.
type Result struct {
	Msg     string // Текст лога
	MsgType string // Тип лога (INFO, COMBAT, SPEECH)
	// В будущем сюда можно добавить:
	// Events []domain.Event // Для анимаций на клиенте
}

// HandlerFunc - это контракт для любой команды (MOVE, ATTACK, etc).
type HandlerFunc func(ctx Context, payload json.RawMessage) (Result, error)

// EmptyResult - вспомогательная функция для пустого успешного ответа
func EmptyResult() Result {
	return Result{}
}

// --- УТИЛИТЫ ---

// FindEntity - ищет сущность по ID среди всех доступных в контексте
func (c Context) FindEntity(id string) *domain.Entity {
	if id == "" {
		return nil
	}
	for _, e := range c.Entities {
		if e.ID == id {
			return e
		}
	}
	return nil
}
