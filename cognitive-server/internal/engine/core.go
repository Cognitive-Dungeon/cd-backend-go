package engine

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine/data"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/systems"
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

	Instance      *ecs2.Instance
	SpellRegistry *data.SpellRegistry // Храним реестр

	// Реактивные системы храним, чтобы они не были собраны GC
	// (хотя EventBus держит ссылки на хендлеры, лучше держать их явно)
	DamageSys *systems.DamageSystem
	DeathSys  *systems.DeathSystem
	ChatSys   *systems.ChatSystem

	// Канал для входящих "задач" от сети
	commandQueue chan func()

	isRunning bool
}

func New(cfg *config.SimulationConfig) *Engine {
	bus := eventbus.New(64)    // TODO: Конкретизировать размер шины
	inst := ecs2.NewInstance() // <--- Создаем мир
	inst.Grid = data.NewGrid(20, 20)

	// Стены (тест)
	for i := data.TileCoord(0); i < 20; i++ {
		inst.Grid.SetTile(data.TilePos{X: i, Y: 0}, enums.TileWall)
		inst.Grid.SetTile(data.TilePos{X: i, Y: 19}, enums.TileWall)
		inst.Grid.SetTile(data.TilePos{X: 0, Y: i}, enums.TileWall)
		inst.Grid.SetTile(data.TilePos{X: 19, Y: i}, enums.TileWall)
	}

	// 1. Загрузка данных
	spellReg := data.NewSpellRegistry()
	// Внимание: путь к assets должен быть корректным относительно точки запуска
	// При запуске из корня проекта: raw_assets/spells.json
	if err := spellReg.LoadFromFile("raw_assets/spells.json"); err != nil {
		logger.Log.Fatalf("Failed to load spells: %v", err)
	}
	logger.Log.Info("✨ Spell Registry loaded")

	// Эти системы не вызываются в Tick(), они реагируют на события.
	damageSys := systems.NewDamageSystem(inst, bus)
	deathSys := systems.NewDeathSystem(inst, bus)
	chatSys := systems.NewChatSystem(inst, bus)

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
	inputCtx := ecs2.NewInputContext(w, e.SpellRegistry)

	systems.InputMoveSystem(inputCtx)  // Move Cmd -> Intent
	systems.InputSpellSystem(inputCtx) // Cast Cmd -> Intent
	inputCtx.Commit()
	w.ClearScope(ecs.ScopeInput) // Удаляем сырые команды

	// 2. LOGIC PHASE
	logicCtx := ecs2.NewLogicContext(w, e.Instance.Grid, e.Bus, e.SpellRegistry)
	// Система Movement: IntentMove -> Position change
	systems.LogicMoveSystem(logicCtx)
	systems.LogicSpellLogic(logicCtx)

	// Здесь будут остальные системы (Combat, Spell и т.д.)

	w.ClearScope(ecs.ScopeLogic) // Удаляем интенты

	// 3. END FRAME
	w.EndFrame() // Удаляем Events
}

func spawnTestEntities(inst *ecs2.Instance) {
	// Игрок
	playerGuid := inst.CreateObject(enums.ObjectTypePlayer)
	inst.NewEntityBuilder(playerGuid).
		WithName("Leeroy").
		WithRender('@', 0x00FF00).
		WithStats(100, 100).
		WithPosition(10, 10).
		WithSpells(1, 2, 4)

	// Манекен
	dummyGuid := inst.CreateObject(enums.ObjectTypeCreature)
	inst.NewEntityBuilder(dummyGuid).
		WithName("Training Dummy").
		WithRender('D', 0xFF0000).
		WithStats(1000, 1000).
		WithPosition(12, 10)

	logger.Log.Infof("Spawned Entities: Player=%s, Dummy=%s", playerGuid, dummyGuid)
}
