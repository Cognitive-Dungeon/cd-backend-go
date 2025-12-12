package dungeon

import "cognitive-server/internal/domain"

// EntityTemplate определяет шаблон для создания сущности
type EntityTemplate struct {
	Name        string
	Type        string
	Symbol      string
	Color       string
	Description string

	// Stats (если есть)
	HP       int
	Strength int
	Gold     int

	// AI behavior
	IsHostile   bool
	Personality string
}

// SpawnEntity создает сущность из шаблона на заданной позиции
func (t EntityTemplate) SpawnEntity(pos domain.Position, level int) domain.Entity {
	entity := domain.Entity{
		ID:    domain.GenerateID(),
		Type:  t.Type,
		Name:  t.Name,
		Pos:   pos,
		Level: level,
		Render: &domain.RenderComponent{
			Symbol: t.Symbol,
			Color:  t.Color,
		},
		Narrative: &domain.NarrativeComponent{
			Description: t.Description,
		},
	}

	// Добавляем Stats если это существо
	if t.HP > 0 {
		entity.Stats = &domain.StatsComponent{
			HP:       t.HP,
			MaxHP:    t.HP,
			Strength: t.Strength,
			Gold:     t.Gold,
		}

		// Добавляем AI компонент
		entity.AI = &domain.AIComponent{
			IsHostile:   t.IsHostile,
			Personality: t.Personality,
			State:       "IDLE",
		}

		// Добавляем зрение и память
		entity.Vision = &domain.VisionComponent{Radius: domain.VisionRadius}
		entity.Memory = &domain.MemoryComponent{ExploredPerLevel: make(map[int]map[int]bool)}
	}

	return entity
}

// --- ВРАГИ ---

var Goblin = EntityTemplate{
	Name:        "Хитрый Гоблин",
	Type:        domain.EntityTypeEnemy,
	Symbol:      "g",
	Color:       "#22C55E",
	Description: "Мелкий пакостный гоблин, воровато оглядывается.",
	HP:          15,
	Strength:    2,
	Gold:        5,
	IsHostile:   true,
	Personality: "Cowardly",
}

var Orc = EntityTemplate{
	Name:        "Свирепый Орк",
	Type:        domain.EntityTypeEnemy,
	Symbol:      "O",
	Color:       "#DC2626",
	Description: "Огромный зеленокожий орк с тяжелой дубиной.",
	HP:          30,
	Strength:    5,
	Gold:        10,
	IsHostile:   true,
	Personality: "Furious",
}

var Troll = EntityTemplate{
	Name:        "Каменный Тролль",
	Type:        domain.EntityTypeEnemy,
	Symbol:      "T",
	Color:       "#78716C",
	Description: "Массивное существо с каменной кожей.",
	HP:          50,
	Strength:    8,
	Gold:        20,
	IsHostile:   true,
	Personality: "Aggressive",
}

// --- NPC (мирные) ---

var Merchant = EntityTemplate{
	Name:        "Торговец",
	Type:        domain.EntityTypeNPC,
	Symbol:      "M",
	Color:       "#FCD34D",
	Description: "Странствующий торговец с тележкой товаров.",
	HP:          20,
	Strength:    1,
	Gold:        100,
	IsHostile:   false,
	Personality: "Friendly",
}

// EnemyTemplates - карта всех доступных врагов
var EnemyTemplates = map[string]EntityTemplate{
	"goblin": Goblin,
	"orc":    Orc,
	"troll":  Troll,
}

// NPCTemplates - карта всех NPC
var NPCTemplates = map[string]EntityTemplate{
	"merchant": Merchant,
}

// --- ПРЕДМЕТЫ ---

// ItemTemplate определяет шаблон для создания предмета-сущности
type ItemTemplate struct {
	Name        string
	Symbol      string
	Color       string
	Description string

	// Item properties
	Category     string
	IsStackable  bool
	AttackSpeed  int
	Damage       int
	Defense      int
	EffectType   string
	EffectValue  int
	IsConsumable bool
	Weight       int
	Value        int

	// Sentient item properties
	IsSentient  bool
	Personality string
	Chattiness  int
}

// SpawnItem создаёт Entity-предмет из шаблона
func (t ItemTemplate) SpawnItem(pos domain.Position, level int) *domain.Entity {
	entity := &domain.Entity{
		ID:    domain.GenerateID(),
		Type:  domain.EntityTypeItem,
		Name:  t.Name,
		Pos:   pos,
		Level: level,
		Render: &domain.RenderComponent{
			Symbol: t.Symbol,
			Color:  t.Color,
		},
		Item: &domain.ItemComponent{
			Category:     t.Category,
			IsStackable:  t.IsStackable,
			StackSize:    1,
			Damage:       t.Damage,
			Defense:      t.Defense,
			EffectType:   t.EffectType,
			EffectValue:  t.EffectValue,
			IsConsumable: t.IsConsumable,
			Weight:       t.Weight,
			Value:        t.Value,
			IsSentient:   t.IsSentient,
			Personality:  t.Personality,
			Chattiness:   t.Chattiness,
		},
	}

	// Для живых предметов добавляем компоненты
	if t.IsSentient {
		entity.Narrative = &domain.NarrativeComponent{
			Description: t.Description,
		}
		entity.AI = &domain.AIComponent{}
	}

	return entity
}

// --- ОРУЖИЕ ---

var IronSword = ItemTemplate{
	Name:        "Железный меч",
	Symbol:      "†",
	Color:       "#C0C0C0",
	Description: "Простой, но надёжный железный меч.",
	Category:    domain.ItemCategoryWeapon,
	Damage:      5,
	Weight:      3,
	Value:       50,
}

var SteelDagger = ItemTemplate{
	Name:        "Стальной кинжал",
	Symbol:      "†",
	Color:       "#E5E7EB",
	Description: "Быстрый и лёгкий кинжал.",
	Category:    domain.ItemCategoryWeapon,
	Damage:      3,
	AttackSpeed: -20, // быстрее на 20 тиков
	Weight:      1,
	Value:       30,
}

var WoodenClub = ItemTemplate{
	Name:        "Деревянная дубина",
	Symbol:      "†",
	Color:       "#78350F",
	Description: "Грубая деревянная дубина.",
	Category:    domain.ItemCategoryWeapon,
	Damage:      4,
	Weight:      2,
	Value:       15,
}

// --- БРОНЯ ---

var LeatherArmor = ItemTemplate{
	Name:        "Кожаная броня",
	Symbol:      "[",
	Color:       "#92400E",
	Description: "Лёгкая кожаная броня.",
	Category:    domain.ItemCategoryArmor,
	Defense:     2,
	Weight:      5,
	Value:       40,
}

var ChainMail = ItemTemplate{
	Name:        "Кольчуга",
	Symbol:      "[",
	Color:       "#9CA3AF",
	Description: "Прочная кольчуга из стальных колец.",
	Category:    domain.ItemCategoryArmor,
	Defense:     5,
	Weight:      10,
	Value:       100,
}

var PlateArmor = ItemTemplate{
	Name:        "Латная броня",
	Symbol:      "[",
	Color:       "#6B7280",
	Description: "Тяжёлая латная броня рыцаря.",
	Category:    domain.ItemCategoryArmor,
	Defense:     8,
	Weight:      20,
	Value:       200,
}

// --- ЗЕЛЬЯ ---

var HealthPotion = ItemTemplate{
	Name:         "Зелье лечения",
	Symbol:       "!",
	Color:        "#DC2626",
	Description:  "Красное зелье, восстанавливающее здоровье.",
	Category:     domain.ItemCategoryPotion,
	EffectType:   "heal",
	EffectValue:  30,
	IsConsumable: true,
	Weight:       0,
	Value:        25,
}

var StrengthPotion = ItemTemplate{
	Name:         "Зелье силы",
	Symbol:       "!",
	Color:        "#CA8A04",
	Description:  "Оранжевое зелье, временно увеличивающее силу.",
	Category:     domain.ItemCategoryPotion,
	EffectType:   "buff_strength",
	EffectValue:  5,
	IsConsumable: true,
	Weight:       0,
	Value:        50,
}

var StaminaPotion = ItemTemplate{
	Name:         "Зелье выносливости",
	Symbol:       "!",
	Color:        "#16A34A",
	Description:  "Зелёное зелье, восстанавливающее выносливость.",
	Category:     domain.ItemCategoryPotion,
	EffectType:   "restore_stamina",
	EffectValue:  50,
	IsConsumable: true,
	Weight:       0,
	Value:        20,
}

// --- ЕДА ---

var Bread = ItemTemplate{
	Name:         "Хлеб",
	Symbol:       "%",
	Color:        "#D97706",
	Description:  "Свежий хлеб.",
	Category:     domain.ItemCategoryFood,
	EffectType:   "restore_stamina",
	EffectValue:  20,
	IsConsumable: true,
	IsStackable:  true,
	Weight:       0,
	Value:        5,
}

var Meat = ItemTemplate{
	Name:         "Мясо",
	Symbol:       "%",
	Color:        "#991B1B",
	Description:  "Сырое мясо.",
	Category:     domain.ItemCategoryFood,
	EffectType:   "restore_stamina",
	EffectValue:  30,
	IsConsumable: true,
	IsStackable:  true,
	Weight:       1,
	Value:        10,
}

// --- РАЗНОЕ ---

var GoldCoin = ItemTemplate{
	Name:        "Золотая монета",
	Symbol:      "$",
	Color:       "#FCD34D",
	Description: "Сверкающая золотая монета.",
	Category:    domain.ItemCategoryMisc,
	IsStackable: true,
	Weight:      0,
	Value:       1,
}

var Torch = ItemTemplate{
	Name:        "Факел",
	Symbol:      "~",
	Color:       "#F59E0B",
	Description: "Горящий факел.",
	Category:    domain.ItemCategoryMisc,
	IsStackable: true,
	Weight:      1,
	Value:       5,
}

// ItemTemplates - карта всех доступных предметов
var ItemTemplates = map[string]ItemTemplate{
	// Оружие
	"iron_sword":   IronSword,
	"steel_dagger": SteelDagger,
	"wooden_club":  WoodenClub,

	// Броня
	"leather_armor": LeatherArmor,
	"chain_mail":    ChainMail,
	"plate_armor":   PlateArmor,

	// Зелья
	"health_potion":   HealthPotion,
	"strength_potion": StrengthPotion,
	"stamina_potion":  StaminaPotion,

	// Еда
	"bread": Bread,
	"meat":  Meat,

	// Разное
	"gold":  GoldCoin,
	"torch": Torch,
}

// --- ЖИВЫЕ ПРЕДМЕТЫ (Для будущей интеграции с LLM) ---

// TODO: Интеграция с микросервисом для генерации диалогов
// Эти предметы имеют личность и будут реагировать на события через WebSocket

var BloodthirstySword = ItemTemplate{
	Name:        "Кровожадный Клинок",
	Symbol:      "†",
	Color:       "#DC2626",
	Description: "Древний меч, жаждущий крови врагов. Шепчет своему владельцу тёмные мысли.",
	Category:    domain.ItemCategoryWeapon,
	Damage:      8,
	Weight:      4,
	Value:       200,
	IsSentient:  true,
	Personality: "sadistic", // Садист - радуется урону
	Chattiness:  7,          // Очень разговорчивый
}

var CowardlyShield = ItemTemplate{
	Name:        "Трусливый Щит",
	Symbol:      "[",
	Color:       "#FCD34D",
	Description: "Щит, который постоянно жалуется и советует убегать.",
	Category:    domain.ItemCategoryArmor,
	Defense:     6,
	Weight:      8,
	Value:       150,
	IsSentient:  true,
	Personality: "cowardly", // Трус - боится опасности
	Chattiness:  8,
}

var GreedyRing = ItemTemplate{
	Name:        "Жадное Кольцо",
	Symbol:      "○",
	Color:       "#F59E0B",
	Description: "Волшебное кольцо, одержимое золотом.",
	Category:    domain.ItemCategoryMisc,
	Weight:      0,
	Value:       300,
	IsSentient:  true,
	Personality: "greedy", // Жадный - реагирует на золото
	Chattiness:  5,
}

var MasochisticArmor = ItemTemplate{
	Name:        "Мазохистские Латы",
	Symbol:      "[",
	Color:       "#6B7280",
	Description: "Тяжёлая броня, которая наслаждается получением урона.",
	Category:    domain.ItemCategoryArmor,
	Defense:     10,
	Weight:      25,
	Value:       250,
	IsSentient:  true,
	Personality: "masochistic", // Мазохист - радуется урону по себе
	Chattiness:  6,
}

// SentientItemTemplates - карта живых предметов (для будущей реализации)
var SentientItemTemplates = map[string]ItemTemplate{
	"bloodthirsty_sword": BloodthirstySword,
	"cowardly_shield":    CowardlyShield,
	"greedy_ring":        GreedyRing,
	"masochistic_armor":  MasochisticArmor,
}
