package domain

// Wait добавляет задержку к следующему действию
func (a *AIComponent) Wait(ticks int) {
	a.NextActionTick += ticks
}

// BecomeHostile переводит моба в режим атаки
func (a *AIComponent) BecomeHostile() {
	a.IsHostile = true
	a.State = AIStateCombat
}

// CalmDown успокаивает моба
func (a *AIComponent) CalmDown() {
	a.IsHostile = false
	a.State = AIStateIdle
}

// IsReady проверяет, настал ли ход (относительно глобального времени)
func (a *AIComponent) IsReady(globalTick int) bool {
	return a.NextActionTick <= globalTick
}
