package domain

// TakeDamage наносит урон. Возвращает true, если цель погибла.
func (s *StatsComponent) TakeDamage(amount int) bool {
	if s.IsDead {
		return false
	}

	// Тут можно будет добавить (amount - s.Defense)
	if amount < 0 {
		amount = 0
	}

	s.HP -= amount

	if s.HP <= 0 {
		s.HP = 0
		s.IsDead = true
		return true
	}
	return false
}

// Heal лечит сущность
func (s *StatsComponent) Heal(amount int) {
	if s.IsDead {
		return // Не лечим трупы! Нет некромантии!
	}
	s.HP += amount
	if s.HP > s.MaxHP {
		s.HP = s.MaxHP
	}
}

// HasStamina проверяет, хватает ли сил
func (s *StatsComponent) HasStamina(cost int) bool {
	return s.Stamina >= cost
}

// SpendStamina тратит силы. Возвращает false, если не хватило.
func (s *StatsComponent) SpendStamina(cost int) bool {
	if s.Stamina < cost {
		return false
	}
	s.Stamina -= cost
	return true
}

// RestoreStamina восстанавливает силы (реген)
func (s *StatsComponent) RestoreStamina(amount int) {
	s.Stamina += amount
	if s.Stamina > s.MaxStamina {
		s.Stamina = s.MaxStamina
	}
}
