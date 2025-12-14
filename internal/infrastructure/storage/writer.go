package storage

import (
	"cognitive-server/internal/domain"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	MagicHeader string = `CDRP` // 4 байта
	Version1    uint32 = 1
)

// ReplayFileHeader — это точное представление заголовка файла в памяти.
// binary.Write умеет писать это целиком, так как тут нет слайсов и строк, только массивы и числа.
type ReplayFileHeader struct {
	Magic       [4]byte // 4 байта
	Version     uint32  // 4 байта
	Seed        int64   // 8 байт
	Timestamp   int64   // 8 байт
	LevelID     int32   // 4 байта
	ActionCount int32   // 4 байта
}

// ActionHeader — заголовок каждой записи действия.
type ActionHeader struct {
	Tick       int32  // 4
	ActionType uint8  // 1
	TokenLen   uint8  // 1
	PayloadLen uint16 // 2
}

type ReplayService struct {
	SaveDir string
}

func NewReplayService(dir string) *ReplayService {
	// Создаем папку если нет
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.Mkdir(dir, 0755)
	}
	return &ReplayService{SaveDir: dir}
}

func (s *ReplayService) Save(session *domain.ReplaySession) error {
	filename := fmt.Sprintf("replay_%d_lvl%d_%d.cdrp", session.Seed, session.LevelID, session.Timestamp)
	path := filepath.Join(s.SaveDir, filename)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// Используем буферизацию для скорости, если записей много,
	// но для простоты пишем напрямую.
	return writeBinary(f, session)
}

func writeBinary(w io.Writer, s *domain.ReplaySession) error {
	// 1. Подготавливаем и пишем ГЛОБАЛЬНЫЙ ЗАГОЛОВОК
	header := ReplayFileHeader{
		Version:     Version1,
		Seed:        s.Seed,
		Timestamp:   s.Timestamp,
		LevelID:     int32(s.LevelID),
		ActionCount: int32(len(s.Actions)),
	}
	copy(header.Magic[:], MagicHeader) // Копируем строку в массив [4]byte

	// ПИШЕМ СТРУКТУРУ ЦЕЛИКОМ
	// Это заменило 6 вызовов binary.Write
	if err := binary.Write(w, binary.LittleEndian, &header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// 2. Пишем действия
	for _, act := range s.Actions {
		tokenBytes := []byte(act.Token)
		if len(tokenBytes) > 255 {
			return fmt.Errorf("token too long: %d", len(tokenBytes))
		}

		payloadLen := len(act.Payload)
		if payloadLen > 65535 {
			return fmt.Errorf("payload too long: %d", payloadLen)
		}

		// Подготавливаем заголовок действия
		actHeader := ActionHeader{
			Tick:       int32(act.Tick),
			ActionType: uint8(act.Action),
			TokenLen:   uint8(len(tokenBytes)),
			PayloadLen: uint16(payloadLen),
		}

		// Пишем заголовок действия одной командой
		if err := binary.Write(w, binary.LittleEndian, &actHeader); err != nil {
			return err
		}

		// Пишем динамические данные (тело)
		if _, err := w.Write(tokenBytes); err != nil {
			return err
		}
		if payloadLen > 0 {
			if _, err := w.Write(act.Payload); err != nil {
				return err
			}
		}
	}

	return nil
}
