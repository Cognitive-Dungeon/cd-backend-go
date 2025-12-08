package api

import "errors"

// Validator - интерфейс, который могут реализовать DTO
type Validator interface {
	Validate() error
}

func (p DirectionPayload) Validate() error {
	if p.Dx == 0 && p.Dy == 0 {
		return errors.New("movement vector cannot be zero")
	}
	if p.Dx < -1 || p.Dx > 1 || p.Dy < -1 || p.Dy > 1 {
		return errors.New("movement step too large")
	}
	return nil
}

func (p EntityPayload) Validate() error {
	if p.TargetID == "" {
		return errors.New("targetId is required")
	}
	return nil
}
