package rules

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine/handlers"
	"cognitive-server/internal/systems/interaction"
	"fmt"
	"math"
)

func init() {
	interaction.Register(StairsRule{})
}

type StairsRule struct{}

func (StairsRule) Name() string { return "UseStairs" }

func (StairsRule) Match(w *domain.GameWorld, actor, target *domain.Entity) bool {
	// ПРОВЕРКА: Есть ли у цели компонент Transition в ECS?
	_, ok := domain.GetComponent[domain.TransitionComponent](w.Components, target.ID)
	return ok
}

func (StairsRule) Apply(ctx handlers.Context, target *domain.Entity) (handlers.Result, error) {
	// 1. Получаем свойства текущей лестницы
	srcTrans, ok := domain.GetComponent[domain.TransitionComponent](ctx.World.Components, target.ID)
	if !ok {
		return handlers.Result{}, fmt.Errorf("component missing")
	}

	targetLevelID := ctx.Actor.Level + int(srcTrans.Direction)

	// Получаем доступ к миру назначения (через сервис)
	// Важно: GameService должен иметь метод GetWorld(id), который загрузит его если надо.
	targetWorld := ctx.Switcher.GetWorld(targetLevelID)
	if targetWorld == nil {
		return handlers.Result{Msg: "Путь ведет в никуда (конец мира)."}, nil
	}

	// 2. ИЩЕМ СООТВЕТСТВИЕ (Constraint Solver)
	var bestPos domain.Position
	found := false
	minDist := math.MaxFloat64

	// Берем хранилище всех переходов того мира
	storage := domain.GetStorage[domain.TransitionComponent](targetWorld.Components)

	for id, destTrans := range storage.Data {
		// CONSTRAINT 1: Тип должен совпадать (Лестница <-> Лестница)
		if destTrans.Type != srcTrans.Type {
			continue
		}

		// CONSTRAINT 2: Направление должно быть противоположным
		// Вверх (-1) ищет Вниз (1). Вниз (1) ищет Вверх (-1).
		if destTrans.Direction != -srcTrans.Direction {
			continue
		}

		// Если мы тут — это валидная точка выхода.
		// Выбираем лучшую по геометрии.
		ent := targetWorld.GetEntity(id)
		if ent == nil {
			continue
		}

		// Эвристика: Ближайшая по X,Y
		dist := target.Pos.DistanceSquaredTo(ent.Pos)
		if float64(dist) < minDist {
			minDist = float64(dist)
			bestPos = ent.Pos
			found = true
		}
	}

	// 3. Fallback (Устойчивость)
	if !found {
		// Если лестницы нет, спускаемся "в пол" (если там не стена)
		if !targetWorld.Map[target.Pos.Y][target.Pos.X].IsWall {
			bestPos = target.Pos
		} else {
			// Крайний случай: ищем любую свободную точку в радиусе
			// TODO: (Тут можно добавить простой спиральный поиск, пока ставим хардкод)
			bestPos = domain.Position{X: targetWorld.Width / 2, Y: targetWorld.Height / 2}
		}
	}

	// 4. Исполнение
	ctx.Switcher.Teleport(ctx.Actor, targetLevelID, bestPos)

	return handlers.Result{
		Msg:     fmt.Sprintf("Вы переходите на уровень %d.", targetLevelID),
		MsgType: "INFO",
	}, nil
}

func init() {
	interaction.Register(StairsRule{})
}
