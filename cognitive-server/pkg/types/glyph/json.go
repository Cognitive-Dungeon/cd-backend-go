package glyph

import (
	"fmt"
)

// ParseGlyphFromJSON парсит глиф из JSON-полей.
//
// Правила:
//   - если char пуст → используется пробел ' '
//   - используется ТОЛЬКО первый байт строки
//   - color ожидается в формате "#RRGGBB" или "RRGGBB"
func ParseGlyphFromJSON(char, color string) (Glyph, error) {
	// --- 1. Символ ---
	src := char
	if src == "" {
		src = "\x20"
	}

	// Берём только первый байт
	ch := src[0]

	// --- 2. Цвет ---
	rgb, err := parseHexColor(color)
	if err != nil {
		return 0, err
	}

	return MakeGlyph(rgb, ch), nil
}

func parseHexColor(s string) (uint32, error) {
	if s == "" {
		return 0, nil // допустимо: по умолчанию чёрный
	}

	if s[0] == '#' {
		s = s[1:]
	}
	if len(s) != 6 {
		return 0, fmt.Errorf("invalid hex color: %q", s)
	}

	var c uint32
	for i := 0; i < 6; i++ {
		c <<= 4
		switch {
		case s[i] >= '0' && s[i] <= '9':
			c |= uint32(s[i] - '0')
		case s[i] >= 'a' && s[i] <= 'f':
			c |= uint32(s[i] - 'a' + 10)
		case s[i] >= 'A' && s[i] <= 'F':
			c |= uint32(s[i] - 'A' + 10)
		default:
			return 0, fmt.Errorf("invalid hex digit %q in color", s[i])
		}
	}

	return c, nil
}
