package dungeon

import (
	"cognitive-server/internal/domain"
	"cognitive-server/pkg/utils"
	"math/rand"
)

// EntityTemplate определяет шаблон для создания сущности
type EntityTemplate struct {
	Name      string
	Type      domain.EntityType
	Render    domain.RenderComponent
	Stats     domain.StatsComponent
	AI        domain.AIComponent
	Narrative domain.NarrativeComponent
}

// SpawnEntity создает сущность из шаблона на заданной позиции
func (t EntityTemplate) SpawnEntity(pos domain.Position, level int, rng *rand.Rand) domain.Entity {
	entity := domain.Entity{
		ID:    utils.GenerateDeterministicID(rng, "e_"),
		Type:  t.Type,
		Name:  t.Name,
		Pos:   pos,
		Level: level,
		Render: &domain.RenderComponent{
			Symbol: t.Render.Symbol,
			Color:  t.Render.Color,
		},
		Narrative: &domain.NarrativeComponent{
			Description: t.Narrative.Description,
		},
	}

	// Добавляем Stats если это существо
	if t.Stats.HP > 0 {
		entity.Stats = &domain.StatsComponent{
			HP:       t.Stats.HP,
			MaxHP:    t.Stats.HP,
			Strength: t.Stats.Strength,
			Gold:     t.Stats.Gold,
		}

		// Добавляем AI компонент
		entity.AI = &domain.AIComponent{
			IsHostile:   t.AI.IsHostile,
			Personality: t.AI.Personality,
			State:       domain.AIStateIdle,
		}

		// Добавляем зрение и память
		entity.Vision = &domain.VisionComponent{Radius: domain.VisionRadius}
		entity.Memory = &domain.MemoryComponent{ExploredPerLevel: make(map[int]map[int]bool)}
	}

	return entity
}

// --- ВРАГИ ---

var Goblin = EntityTemplate{
	Name: "Хитрый Гоблин",
	Type: domain.EntityTypeEnemy,
	Render: domain.RenderComponent{
		Symbol: 'g',
		Color:  "#22C55E",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Мелкий пакостный гоблин, воровато оглядывается.",
	},
	Stats: domain.StatsComponent{
		HP:       15,
		Strength: 2,
		Gold:     5,
	},
	AI: domain.AIComponent{
		IsHostile:   true,
		Personality: "Cowardly",
	},
}

var Orc = EntityTemplate{
	Name: "Свирепый Орк",
	Type: domain.EntityTypeEnemy,
	Render: domain.RenderComponent{
		Symbol: 'O',
		Color:  "#DC2626",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Огромный зеленокожий орк с тяжелой дубиной.",
	},
	Stats: domain.StatsComponent{
		HP:       30,
		Strength: 5,
		Gold:     10,
	},
	AI: domain.AIComponent{
		IsHostile:   true,
		Personality: "Furious",
	},
}

var Troll = EntityTemplate{
	Name: "Каменный Тролль",
	Type: domain.EntityTypeEnemy,
	Render: domain.RenderComponent{
		Symbol: 'T',
		Color:  "#78716C",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Массивное существо с каменной кожей.",
	},
	Stats: domain.StatsComponent{
		HP:       50,
		Strength: 8,
		Gold:     20,
	},
	AI: domain.AIComponent{
		IsHostile:   true,
		Personality: "Aggressive",
	},
}

// --- NPC (мирные) ---

var Merchant = EntityTemplate{
	Name: "Торговец",
	Type: domain.EntityTypeNPC,
	Render: domain.RenderComponent{
		Symbol: 'M',
		Color:  "#FCD34D",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Странствующий торговец с тележкой товаров.",
	},
	Stats: domain.StatsComponent{
		HP:       20,
		Strength: 1,
		Gold:     100,
	},
	AI: domain.AIComponent{
		IsHostile:   false,
		Personality: "Friendly",
	},
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
	Name      string
	Render    domain.RenderComponent
	Narrative domain.NarrativeComponent

	// Item properties
	Properties domain.ItemComponent
}

// SpawnItem создаёт Entity-предмет из шаблона
func (t ItemTemplate) SpawnItem(pos domain.Position, level int, rng *rand.Rand) *domain.Entity {
	entity := &domain.Entity{
		ID:    utils.GenerateDeterministicID(rng, "ei_"),
		Type:  domain.EntityTypeItem,
		Name:  t.Name,
		Pos:   pos,
		Level: level,
		Render: &domain.RenderComponent{
			Symbol: t.Render.Symbol,
			Color:  t.Render.Color,
		},
		Item: &domain.ItemComponent{

			Category:     t.Properties.Category,
			IsStackable:  t.Properties.IsStackable,
			StackSize:    1,
			Damage:       t.Properties.Damage,
			Defense:      t.Properties.Defense,
			EffectType:   t.Properties.EffectType,
			EffectValue:  t.Properties.EffectValue,
			IsConsumable: t.Properties.IsConsumable,
			Weight:       t.Properties.Weight,
			Price:        t.Properties.Price,
			IsSentient:   t.Properties.IsSentient,
			Personality:  t.Properties.Personality,
			Chattiness:   t.Properties.Chattiness,
		},
	}

	// Для живых предметов добавляем компоненты
	if t.Properties.IsSentient {
		entity.Narrative = &domain.NarrativeComponent{
			Description: t.Narrative.Description,
		}
		entity.AI = &domain.AIComponent{}
	}

	return entity
}

// --- ОРУЖИЕ ---

var IronSword = ItemTemplate{
	Name: "Железный меч",
	Render: domain.RenderComponent{
		Symbol: ')',
		Color:  "#C0C0C0",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Простой, но надёжный железный меч.",
	},
	Properties: domain.ItemComponent{
		Category: domain.ItemCategoryWeapon,
		Damage:   5,
		Weight:   3,
		Price:    50,
	},
}

var SteelDagger = ItemTemplate{
	Name: "Стальной кинжал",
	Render: domain.RenderComponent{
		Symbol: ')',
		Color:  "#E5E7EB",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Быстрый и лёгкий кинжал.",
	},
	Properties: domain.ItemComponent{
		Category:    domain.ItemCategoryWeapon,
		Damage:      3,
		AttackSpeed: -20, // быстрее на 20 тиков
		Weight:      1,
		Price:       30,
	},
}

var WoodenClub = ItemTemplate{
	Name: "Деревянная дубина",
	Render: domain.RenderComponent{
		Symbol: ')',
		Color:  "#78350F",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Грубая деревянная дубина.",
	},
	Properties: domain.ItemComponent{
		Category: domain.ItemCategoryWeapon,
		Damage:   4,
		Weight:   2,
		Price:    15,
	},
}

// --- БРОНЯ ---

var LeatherArmor = ItemTemplate{
	Name: "Кожаная броня",
	Render: domain.RenderComponent{
		Symbol: '[',
		Color:  "#92400E",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Лёгкая кожаная броня.",
	},
	Properties: domain.ItemComponent{
		Category: domain.ItemCategoryArmor,
		Defense:  2,
		Weight:   5,
		Price:    40,
	},
}

var ChainMail = ItemTemplate{
	Name: "Кольчуга",
	Render: domain.RenderComponent{
		Symbol: '[',
		Color:  "#9CA3AF",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Прочная кольчуга из стальных колец.",
	},
	Properties: domain.ItemComponent{
		Category: domain.ItemCategoryArmor,
		Defense:  5,
		Weight:   10,
		Price:    100,
	},
}

var PlateArmor = ItemTemplate{
	Name: "Латная броня",
	Render: domain.RenderComponent{
		Symbol: '[',
		Color:  "#6B7280",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Тяжёлая латная броня рыцаря.",
	},
	Properties: domain.ItemComponent{
		Category: domain.ItemCategoryArmor,
		Defense:  8,
		Weight:   20,
		Price:    200,
	},
}

// --- ЗЕЛЬЯ ---

var HealthPotion = ItemTemplate{
	Name: "Зелье лечения",
	Render: domain.RenderComponent{
		Symbol: '!',
		Color:  "#DC2626",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Красное зелье, восстанавливающее здоровье.",
	},
	Properties: domain.ItemComponent{
		Category:     domain.ItemCategoryPotion,
		EffectType:   "heal",
		EffectValue:  30,
		IsConsumable: true,
		Weight:       0,
		Price:        25,
	},
}

var StrengthPotion = ItemTemplate{
	Name: "Зелье силы",
	Render: domain.RenderComponent{
		Symbol: '!',
		Color:  "#CA8A04",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Оранжевое зелье, временно увеличивающее силу.",
	},
	Properties: domain.ItemComponent{
		Category:     domain.ItemCategoryPotion,
		EffectType:   "buff_strength",
		EffectValue:  5,
		IsConsumable: true,
		Weight:       0,
		Price:        50,
	},
}

var StaminaPotion = ItemTemplate{
	Name: "Зелье выносливости",
	Render: domain.RenderComponent{
		Symbol: '!',
		Color:  "#16A34A",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Зелёное зелье, восстанавливающее выносливость.",
	},
	Properties: domain.ItemComponent{
		Category:     domain.ItemCategoryPotion,
		EffectType:   "restore_stamina",
		EffectValue:  50,
		IsConsumable: true,
		Weight:       0,
		Price:        20,
	},
}

// --- ЕДА ---

var Bread = ItemTemplate{
	Name: "Хлеб",
	Render: domain.RenderComponent{
		Symbol: '%',
		Color:  "#D97706",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Свежий хлеб.",
	},
	Properties: domain.ItemComponent{
		Category:     domain.ItemCategoryFood,
		EffectType:   "restore_stamina",
		EffectValue:  20,
		IsConsumable: true,
		IsStackable:  true,
		Weight:       0,
		Price:        5,
	},
}

var Meat = ItemTemplate{
	Name: "Мясо",
	Render: domain.RenderComponent{
		Symbol: '%',
		Color:  "#991B1B",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Сырое мясо.",
	},
	Properties: domain.ItemComponent{
		Category:     domain.ItemCategoryFood,
		EffectType:   "restore_stamina",
		EffectValue:  30,
		IsConsumable: true,
		IsStackable:  true,
		Weight:       1,
		Price:        10,
	},
}

// --- РАЗНОЕ ---

var GoldCoin = ItemTemplate{
	// TODO: Повесить эффект добавления золота актору при подборе
	Name: "Золотая монета",
	Render: domain.RenderComponent{
		Symbol: '$',
		Color:  "#FCD34D",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Сверкающая золотая монета.",
	},
	Properties: domain.ItemComponent{
		Category:    domain.ItemCategoryMisc,
		IsStackable: true,
		Weight:      0,
		Price:       1,
	},
}

var Torch = ItemTemplate{
	// TODO: Повесить эффект расширения FOV и таймер прогорания
	Name: "Факел",
	Render: domain.RenderComponent{
		Symbol: '~',
		Color:  "#F59E0B",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Горящий факел.",
	},
	Properties: domain.ItemComponent{
		Category:    domain.ItemCategoryMisc,
		IsStackable: true,
		Weight:      1,
		Price:       5,
	},
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
	Name: "Кровожадный Клинок",
	Render: domain.RenderComponent{
		Symbol: ')',
		Color:  "#DC2626",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Древний меч, жаждущий крови врагов. Шепчет своему владельцу тёмные мысли.",
	},
	Properties: domain.ItemComponent{
		Category:    domain.ItemCategoryWeapon,
		Damage:      8,
		Weight:      4,
		Price:       200,
		IsSentient:  true,
		Personality: "sadistic", // Садист - радуется урону
		Chattiness:  7,          // Очень разговорчивый
	},
}

var CowardlyShield = ItemTemplate{
	Name: "Трусливый Щит",
	Render: domain.RenderComponent{
		Symbol: '[',
		Color:  "#FCD34D",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Щит, который постоянно жалуется и советует убегать.",
	},
	Properties: domain.ItemComponent{
		Category:    domain.ItemCategoryArmor,
		Defense:     6,
		Weight:      8,
		Price:       150,
		IsSentient:  true,
		Personality: "cowardly", // Трус - боится опасности
		Chattiness:  8,
	},
}

var GreedyRing = ItemTemplate{
	Name: "Жадное Кольцо",
	Render: domain.RenderComponent{
		Symbol: '=',
		Color:  "#F59E0B",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Волшебное кольцо, одержимое золотом.",
	},
	Properties: domain.ItemComponent{
		Category:    domain.ItemCategoryMisc,
		Weight:      0,
		Price:       300,
		IsSentient:  true,
		Personality: "greedy", // Жадный - реагирует на золото
		Chattiness:  5,
	},
}

var MasochisticArmor = ItemTemplate{
	Name: "Мазохистские Латы",
	Render: domain.RenderComponent{
		Symbol: '[',
		Color:  "#6B7280",
	},
	Narrative: domain.NarrativeComponent{
		Description: "Тяжёлая броня, которая наслаждается получением урона.",
	},
	Properties: domain.ItemComponent{
		Category:    domain.ItemCategoryArmor,
		Defense:     10,
		Weight:      25,
		Price:       250,
		IsSentient:  true,
		Personality: "masochistic", // Мазохист - радуется урону по себе
		Chattiness:  6,
	},
}

// SentientItemTemplates - карта живых предметов (для будущей реализации)
var SentientItemTemplates = map[string]ItemTemplate{
	"bloodthirsty_sword": BloodthirstySword,
	"cowardly_shield":    CowardlyShield,
	"greedy_ring":        GreedyRing,
	"masochistic_armor":  MasochisticArmor,
}

// LootTable - автоматический список ключей всех обычных предметов.
// Заполняется при старте программы.
var LootTable []string

func init() {
	for key := range ItemTemplates {
		LootTable = append(LootTable, key)
	}
}
