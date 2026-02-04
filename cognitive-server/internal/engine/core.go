package engine

import (
	"cognitive-server/internal/config"
	"cognitive-server/internal/core/types/enums"
	"cognitive-server/internal/engine/data"
	ecs2 "cognitive-server/internal/engine/model"
	"cognitive-server/internal/engine/model/systems"
	"cognitive-server/pkg/ecs"
	"cognitive-server/pkg/eventbus"
	"cognitive-server/pkg/geo"
	"cognitive-server/pkg/logger"
	"cognitive-server/pkg/worldmap"
	"cognitive-server/pkg/worldmap/materials"
	"cognitive-server/pkg/worldmap/storage/tiled"
	"context"
	"encoding/json"
	"os"
	"time"
)

const TickRate = 50 * time.Millisecond // 20Hz или 20 TPS

type Engine struct {
	cfg *config.SimulationConfig
	Bus *eventbus.EventBus

	Instance         *ecs2.Instance
	SpellRegistry    *data.SpellRegistry // Храним реестр
	MaterialRegistry *materials.MaterialRegistry

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

	// --- ЗАГРУЗКА КАРТЫ ---
	const (
		//TODO: В продакшене пути должны приходить из конфига
		mapPath = "cognitive-tools/tiled/demo_map.tmj"
		matPath = "assets/materials.json"
	)
	// 1. Загрузка Реестра Материалов (Visuals & Meta)
	// Используется для Snapshot (клиент) и AI
	matReg, err := materials.LoadRegistryFromFile(matPath)
	if err != nil {
		logger.Log.Errorf("Failed to load materials registry: %v. Using empty fallback.", err)
		matReg = materials.NewRegistry()
	} else {
		logger.Log.Info("🎨 Material Registry loaded")
	}

	// 2. Загрузка Физической Карты (Physics & Logic)
	// Используется для коллизий и навигации
	if err := loadMapIntoWorld(inst.WorldMap, mapPath, matPath); err != nil {
		logger.Log.Errorf("Failed to load map: %v. World will be empty.", err)
	} else {
		logger.Log.Info("🌍 World Map loaded successfully")
	}

	// 3. Загрузка Спеллов
	spellReg := data.NewSpellRegistry()
	if err := spellReg.LoadFromFile("raw_assets/spells.json"); err != nil {
		logger.Log.Fatalf("Failed to load spells: %v", err)
	}
	logger.Log.Info("✨ Spell Registry loaded")

	// Системы
	damageSys := systems.NewDamageSystem(inst, bus)
	deathSys := systems.NewDeathSystem(inst, bus)
	chatSys := systems.NewChatSystem(inst, bus)

	// --- ТЕСТОВЫЙ СПАВН ---
	spawnTestEntities(inst)

	return &Engine{
		cfg:              cfg,
		Bus:              bus,
		Instance:         inst,
		SpellRegistry:    spellReg,
		MaterialRegistry: matReg,
		DamageSys:        damageSys,
		DeathSys:         deathSys,
		ChatSys:          chatSys,
		commandQueue:     make(chan func(), 1024),
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

// Хелпер для загрузки палитры и карты (упрощенная версия из inspector)
func loadMapIntoWorld(w *worldmap.World, mapPath, matPath string) error {
	// 1. Load Materials to build Palette
	matData, err := os.ReadFile(matPath)
	if err != nil {
		return err
	}

	// Минимальная структура для чтения JSON
	type matJson struct {
		GID        int                 `json:"gid"`
		MaterialID worldmap.MaterialID `json:"material_id"`
		Flags      worldmap.TileFlag   `json:"flags"`
		Variant    uint8               `json:"variant"`
	}
	var mats []matJson
	if err := json.Unmarshal(matData, &mats); err != nil {
		return err
	}

	palette := tiled.Palette{}
	for _, m := range mats {
		palette[m.GID] = tiled.TileDef{MaterialID: m.MaterialID, Flags: m.Flags, Variant: m.Variant}
	}

	// 2. Load Tiled Map
	store, err := tiled.NewStore(mapPath, palette, geo.Pos(0, 0, 0))
	if err != nil {
		return err
	}

	// 3. Inject into World
	loader := worldmap.NewLoader(w, store)
	// Грузим центр (4x4 чанка для примера)
	loaded, err := loader.LoadRegionChunkCenter(geo.Pos(0, 0, 0), 2)
	if err != nil {
		return err
	}
	logger.Log.Infof("Loaded %d chunks from %s", loaded, mapPath)

	return nil
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
	logicCtx := ecs2.NewLogicContext(w, e.Instance.WorldMap, e.Bus, e.SpellRegistry)
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
