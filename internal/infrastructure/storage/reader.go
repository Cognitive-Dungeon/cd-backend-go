package storage

import (
	"cognitive-server/internal/domain"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func (s *ReplayService) Load(path string) (*domain.ReplaySession, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return readBinary(f)
}

func readBinary(r io.Reader) (*domain.ReplaySession, error) {
	// 1. Читаем заголовок
	var header ReplayFileHeader
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Валидация
	if string(header.Magic[:]) != MagicHeader {
		return nil, fmt.Errorf("invalid magic")
	}
	if header.Version != Version1 {
		return nil, fmt.Errorf("unsupported version: %d", header.Version)
	}

	// Заполняем сессию
	session := &domain.ReplaySession{
		Seed:      header.Seed,
		Timestamp: header.Timestamp,
		LevelID:   int(header.LevelID),
		Actions:   make([]domain.ReplayAction, header.ActionCount),
	}

	// 2. Читаем действия
	for i := 0; i < int(header.ActionCount); i++ {
		// Читаем заголовок действия ЦЕЛИКОМ
		var ah ActionHeader
		if err := binary.Read(r, binary.LittleEndian, &ah); err != nil {
			return nil, err
		}

		act := domain.ReplayAction{
			Tick:   int(ah.Tick),
			Action: domain.ActionType(ah.ActionType),
		}

		// Читаем хвосты (строки и байты)
		tokenBuf := make([]byte, ah.TokenLen)
		if _, err := io.ReadFull(r, tokenBuf); err != nil {
			return nil, err
		}
		act.Token = string(tokenBuf)

		if ah.PayloadLen > 0 {
			act.Payload = make([]byte, ah.PayloadLen)
			if _, err := io.ReadFull(r, act.Payload); err != nil {
				return nil, err
			}
		}

		session.Actions[i] = act
	}

	return session, nil
}
