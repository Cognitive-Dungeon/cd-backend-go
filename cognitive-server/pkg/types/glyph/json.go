package glyph

import (
	"fmt"
	"unicode/utf8"
)

// ParseGlyphFromJSON парсит глиф из строковых значений JSON.
// charStr: может быть "☺" (U+263A) или "A". Должен быть в таблице CP437.
// colorStr: "#RRGGBB"
func ParseGlyphFromJSON(charStr, colorStr string) (Glyph, error) {
	// 1. Парсинг цвета
	rgb, err := parseHexColor(colorStr)
	if err != nil {
		return 0, err
	}

	// 2. Парсинг символа
	if charStr == "" {
		return MakeGlyph(rgb, ' '), nil
	}

	r, size := utf8.DecodeRuneInString(charStr)
	if r == utf8.RuneError {
		return 0, fmt.Errorf("invalid utf8 sequence")
	}
	if size != len(charStr) {
		return 0, fmt.Errorf("glyph char must be exactly one rune, got %q", charStr)
	}

	// Попытка найти в таблице CP437
	b, ok := unicodeToCp437[r]
	if !ok {
		// Fallback: Если символа нет в CP437, но он ASCII (например, тильда ~ или ^)
		// CP437 почти совпадает с ASCII, но есть исключения.
		if r >= 32 && r <= 126 {
			return MakeGlyph(rgb, byte(r)), nil
		}
		return 0, fmt.Errorf("rune %q (U+%04X) is not supported in CP437 palette", charStr, r)
	}

	return MakeGlyph(rgb, b), nil
}

// Хелпер для цвета
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
