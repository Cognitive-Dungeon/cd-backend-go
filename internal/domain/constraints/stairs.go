package constraints

import (
	"cognitive-server/internal/domain"
	"cognitive-server/internal/engine"
)

type IsStairsUp struct{}

func (IsStairsUp) Match(e *domain.Entity, _ *engine.Instance) bool {
	return e.Type == domain.EntityTypeExit && e.StairDirection == domain.StairDirectionUp
}

type IsStairsDown struct{}

func (IsStairsDown) Match(e *domain.Entity, _ *engine.Instance) bool {
	return e.Type == domain.EntityTypeExit && e.StairDirection == domain.StairDirectionDown
}
