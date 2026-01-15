package ecs

// EntityID представляет собой уникальный идентификатор сущности.
// Это 64-битное число, содержащее индекс (для доступа к массивам)
// и поколение (для защиты от ABA-проблемы).
//
// Битовая раскладка совместима с ObjectGuid:
// [ Shard (8) | Type (8) | Generation (16) | Index (32) ]
type EntityID uint64

const (
	// Маски для извлечения частей EntityID.
	maskIndex = 0xFFFFFFFF
	maskGen   = 0xFFFF

	// Сдвиг поколения.
	shiftGen = 32
)

// NilEntityID обозначает нулевой или некорректный идентификатор.
const NilEntityID EntityID = 0

// Index возвращает индекс сущности в массивах данных (младшие 32 бита).
// Используется внутри Storage для O(1) доступа.
func (id EntityID) Index() uint32 {
	return uint32(id & maskIndex)
}

// Generation возвращает версию сущности (16 бит).
// Используется для проверки валидности ссылки (ABA protection).
func (id EntityID) Generation() uint16 {
	return uint16((id >> shiftGen) & maskGen)
}

// IsNil проверяет, является ли EntityID пустым.
func (id EntityID) IsNil() bool {
	return id == 0
}

// Scope определяет время жизни компонента.
// Используется для автоматической очистки временных данных (Request/Intent/Event).
type Scope uint8

const (
	// ScopePersistent — компонент живет вечно (State).
	// Пример: Position, Health, Name, Spellbook.
	ScopePersistent Scope = 0

	// ScopeInput — компонент живет только на этапе обработки ввода.
	// Очищается после SystemInput.
	// Пример: CmdMove, CmdCast (сырые команды).
	ScopeInput Scope = 1 << 0

	// ScopeLogic — компонент живет на этапе игровой логики.
	// Очищается после применения механик (SystemMove, SystemCombat).
	// Пример: IntentMove, IntentAttack (валидированные намерения).
	ScopeLogic Scope = 1 << 1

	// ScopeFrame — компонент живет до самого конца кадра.
	// Очищается последним (после отправки данных клиентам).
	// Пример: EventUnitMoved, EventDamageTaken (события для VFX/UI/Сети).
	ScopeFrame Scope = 1 << 2
)
