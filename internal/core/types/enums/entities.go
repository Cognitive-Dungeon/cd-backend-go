package enums

import "strings"

// ObjectType определяет конкретный тип игрового объекта.
//
// ObjectType — это Masked-style TypeID:
//   - у каждого объекта ровно ОДИН ObjectType
//   - значение хранится в ObjectGuid
//   - используется для базовой идентификации объекта
//
// Для семантических проверок (Object, Enemy, Item и т.п.)
// используется ObjectTypeMask.
type ObjectType uint8

const (
	// ObjectTypeNone обозначает отсутствие типа
	// или неинициализированное значение.
	ObjectTypeNone ObjectType = iota

	// ObjectTypePlayer — игровой персонаж (игрок).
	ObjectTypePlayer

	// ObjectTypeCreature — неигровое существо (NPC, mob).
	ObjectTypeCreature

	// ObjectTypePet — питомец или призванное существо.
	ObjectTypePet

	// ObjectTypeItem — предмет.
	ObjectTypeItem

	// ObjectTypeContainer — контейнер (сумка, сундук).
	ObjectTypeContainer

	// ObjectTypeGameObject — объект мира
	// (дверь, выход, рычаг, интерактивный объект).
	ObjectTypeGameObject

	// ObjectTypeDynamicObject — временный динамический объект
	// (визуальные эффекты, AoE-зоны, spell objects).
	ObjectTypeDynamicObject

	// ObjectTypeCorpse — труп существа или игрока.
	ObjectTypeCorpse
)

// ObjectTypeMask описывает семантические категории объектов.
//
// ObjectTypeMask используется для игровой логики:
//   - проверки принадлежности к категориям
//   - фильтрации объектов
//   - принятия решений (атака, взаимодействие и т.д.)
//
// Один ObjectType может соответствовать нескольким маскам.
type ObjectTypeMask uint32

const (
	// TypeMaskNone обозначает отсутствие категории.
	TypeMaskNone ObjectTypeMask = 0

	// Базовые категории объектов.
	TypeMaskObject ObjectTypeMask = 1 << iota
	TypeMaskUnit
	TypeMaskItem
	TypeMaskWorldObject

	// Категории юнитов.
	TypeMaskPlayer
	TypeMaskCreature
	TypeMaskPet
	TypeMaskEnemy
	TypeMaskNPC

	// Категории объектов мира.
	TypeMaskExit
	TypeMaskDynamic
	TypeMaskCorpse
)

// objectTypeMasks связывает ObjectType с соответствующими семантическими масками.
//
// Это ключевой элемент WoW-подхода:
//   - ObjectType задаёт конкретный тип
//   - ObjectTypeMask описывает поведение и свойства
var objectTypeMasks = map[ObjectType]ObjectTypeMask{
	ObjectTypePlayer: TypeMaskObject |
		TypeMaskUnit |
		TypeMaskPlayer |
		TypeMaskCreature,

	ObjectTypeCreature: TypeMaskObject |
		TypeMaskUnit |
		TypeMaskCreature |
		TypeMaskNPC |
		TypeMaskEnemy,

	ObjectTypePet: TypeMaskObject |
		TypeMaskUnit |
		TypeMaskCreature |
		TypeMaskPet,

	ObjectTypeItem: TypeMaskObject |
		TypeMaskItem,

	ObjectTypeContainer: TypeMaskObject |
		TypeMaskItem,

	ObjectTypeGameObject: TypeMaskObject |
		TypeMaskWorldObject |
		TypeMaskExit,

	ObjectTypeDynamicObject: TypeMaskObject |
		TypeMaskWorldObject |
		TypeMaskDynamic,

	ObjectTypeCorpse: TypeMaskObject |
		TypeMaskCorpse,
}

// Mask возвращает семантическую маску, соответствующую ObjectType.
//
// Маска используется для проверки свойств объекта
// без привязки к конкретному типу.
func (t ObjectType) Mask() ObjectTypeMask {
	return objectTypeMasks[t]
}

// Is проверяет, принадлежит ли ObjectType указанной категории.
//
// Пример:
//
//	if t.Is(TypeMaskEnemy) {
//	    // объект является врагом
//	}
func (t ObjectType) Is(mask ObjectTypeMask) bool {
	return t.Mask()&mask != 0
}

// Строковые представления ObjectType.
//
// Используются для логирования, отладки и конфигурационных файлов.
var objectTypeToString = map[ObjectType]string{
	ObjectTypePlayer:        "PLAYER",
	ObjectTypeCreature:      "CREATURE",
	ObjectTypePet:           "PET",
	ObjectTypeItem:          "ITEM",
	ObjectTypeContainer:     "CONTAINER",
	ObjectTypeGameObject:    "GAMEOBJECT",
	ObjectTypeDynamicObject: "DYNAMICOBJECT",
	ObjectTypeCorpse:        "CORPSE",
}

var objectTypeStringToType = map[string]ObjectType{
	"PLAYER":        ObjectTypePlayer,
	"CREATURE":      ObjectTypeCreature,
	"PET":           ObjectTypePet,
	"ITEM":          ObjectTypeItem,
	"CONTAINER":     ObjectTypeContainer,
	"GAMEOBJECT":    ObjectTypeGameObject,
	"DYNAMICOBJECT": ObjectTypeDynamicObject,
	"CORPSE":        ObjectTypeCorpse,
}

// String возвращает строковое представление ObjectType.
//
// Предназначено для логов и отладки.
func (t ObjectType) String() string {
	if v, ok := objectTypeToString[t]; ok {
		return v
	}
	return "UNKNOWN"
}

// ParseObjectType преобразует строковое значение в ObjectType.
//
// Используется при загрузке конфигураций, шаблонов и данных из файлов.
// Регистр символов не имеет значения.
func ParseObjectType(s string) ObjectType {
	upper := strings.ToUpper(s)
	if v, ok := objectTypeStringToType[upper]; ok {
		return v
	}
	return ObjectTypeNone
}
