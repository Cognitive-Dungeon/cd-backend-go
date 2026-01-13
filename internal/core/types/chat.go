package types

// ChatType — тип сообщения чата.
type ChatType uint8

const (
	ChatTypeNone    ChatType = iota
	ChatTypeSystem           // Системное сообщение
	ChatTypeSay              // Обычная речь (радиус)
	ChatTypeYell             // Крик (большой радиус)
	ChatTypeWhisper          // Личное сообщение
	ChatTypeEmote            // Эмоция
)
