package engine

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/core/types"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/logger"
	"context"
	"time"
)

const TickRate = 50 * time.Millisecond // 20Hz или 20 TPS

type Engine struct {
	cfg *config.SimulationConfig
	Bus *eventbus.EventBus

	Instance      *Instance
	SpellRegistry *SpellRegistry // Храним реестр

	// Реактивные системы храним, чтобы они не были собраны GC
	// (хотя EventBus держит ссылки на хендлеры, лучше держать их явно)
	DamageSys *DamageSystem
	DeathSys  *DeathSystem
	ChatSys   *ChatSystem

	// Канал для входящих "задач" от сети
	commandQueue chan func()

	isRunning bool
}

func New(cfg *config.SimulationConfig) *Engine {
	bus := eventbus.New(64) // TODO: Конкретизировать размер шины
	inst := NewInstance()   // <--- Создаем мир
	inst.Grid = NewGrid(20, 20)

	// Стены (тест)
	for i := TileCoord(0); i < 20; i++ {
		inst.Grid.SetTile(TilePos{X: i, Y: 0}, enums.TileWall)
		inst.Grid.SetTile(TilePos{X: i, Y: 19}, enums.TileWall)
		inst.Grid.SetTile(TilePos{X: 0, Y: i}, enums.TileWall)
		inst.Grid.SetTile(TilePos{X: 19, Y: i}, enums.TileWall)
	}

	// 1. Загрузка данных
	spellReg := NewSpellRegistry()
	// Внимание: путь к assets должен быть корректным относительно точки запуска
	// При запуске из корня проекта: assets/spells.json
	if err := spellReg.LoadFromFile("assets/spells.json"); err != nil {
		logger.Log.Fatalf("Failed to load spells: %v", err)
	}
	logger.Log.Info("✨ Spell Registry loaded")

	// Эти системы не вызываются в Tick(), они реагируют на события.
	damageSys := NewDamageSystem(inst, bus)
	deathSys := NewDeathSystem(inst, bus)
	chatSys := NewChatSystem(inst, bus)

	// --- ТЕСТОВЫЙ СПАВН ---
	spawnTestEntities(inst)

	return &Engine{
		cfg:           cfg,
		Bus:           bus,
		Instance:      inst,
		SpellRegistry: spellReg,
		DamageSys:     damageSys,
		DeathSys:      deathSys,
		ChatSys:       chatSys,
		commandQueue:  make(chan func(), 1024),
	}
}

func (e *Engine) Run() {
	logger.Log.Info("⚙️  Engine: Loop started")
	e.isRunning = true

	ticker := time.NewTicker(TickRate)
	defer ticker.Stop()

	for e.isRunning {
		<-ticker.C
		e.Tick()
	}
	logger.Log.Info("⚙️  Engine: Stopped")
}

func (e *Engine) Stop(ctx context.Context) {
	e.isRunning = false

}

// Метод для безопасного выполнения кода в основном потоке (вызывается из Client)
func (e *Engine) PushCommand(cmd func()) {
	select {
	case e.commandQueue <- cmd:
	default:
		logger.Log.Warn("Engine command queue full!")
	}
}

// Tick — один кадр симуляции.
// Выполняется строго в одной горутине.
func (e *Engine) Tick() {
	w := e.Instance.World
	// 1. Разгребаем очередь команд (Non-blocking drain)
	// Мы выполняем все накопившиеся команды за раз
	// 1. INPUT PHASE
	// Обрабатываем очередь команд (Network -> CmdMove)
loop:
	for {
		select {
		case cmd := <-e.commandQueue:
			cmd() // Выполняем функцию (например, спавн игрока или обработку движения)
		default:
			// Очередь пуста, выходим из цикла обработки команд
			break loop
		}
	}

	SystemInput(w)                       // Move Cmd -> Intent
	SystemSpellInput(w, e.SpellRegistry) // Cast Cmd -> Intent
	w.ClearScope(ecs.ScopeInput)         // Удаляем сырые команды

	// 2. LOGIC PHASE
	// Система Movement: IntentMove -> Position change
	SystemMovement(w, e.Instance.Grid, e.Bus)
	SystemSpellLogic(w, e.SpellRegistry, e.Bus)

	// Здесь будут остальные системы (Combat, Spell и т.д.)

	w.ClearScope(ecs.ScopeLogic) // Удаляем интенты

	// 3. END FRAME
	w.EndFrame() // Удаляем Events
}

func spawnTestEntities(inst *Instance) {
	// Игрок
	playerGuid := inst.CreateObject(enums.ObjectTypePlayer)
	inst.NewEntityBuilder(playerGuid).
		WithName(NameComponent{Name: "Leeroy"}).
		WithRender(types.MakeGlyph(0x00FF00, '@')).
		WithStats(StatsComponent{Health: 100, MaxHealth: 100, Mana: 100, MaxMana: 100}).
		WithPosition(PositionComponent{types.TilePos{X: 10, Y: 10}}).
		WithSpells(SpellbookComponent{
			KnownSpells: []uint32{1, 2, 4}, // Attack, Fireball, Blink
			Cooldowns:   make(map[uint32]float64),
		})

	// Манекен
	dummyGuid := inst.CreateObject(enums.ObjectTypeCreature)
	inst.NewEntityBuilder(dummyGuid).
		WithName(NameComponent{Name: "Training Dummy"}).
		WithRender(types.MakeGlyph(0xFF0000, 'D')).
		WithStats(StatsComponent{Health: 1000, MaxHealth: 1000}).
		WithPosition(PositionComponent{types.TilePos{X: 12, Y: 10}})

	logger.Log.Infof("Spawned Entities: Player=%s, Dummy=%s", playerGuid, dummyGuid)
}
