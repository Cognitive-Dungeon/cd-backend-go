package glyph

// Glyph представляет упакованное представление цветного символа.
// Использует 32 бита (uint32) для хранения в формате:
//
//	[0:8] - символ (8 бит = 1 байт) - маска 0xFF
//	[8:32] - RGB-цвет (24 бита = 3 байта) - маска 0xFFFFFF
type Glyph uint32

// Константы для битовых операций с Glyph
const (
	// Размеры полей в битах
	bitsChar  = 8  // Символ - 8 бит (0-255)
	bitsColor = 24 // Цвет - 24 бита (RGB)

	// Сдвиги для упаковки/распаковки
	shiftColor = bitsChar // Смещение для записи/чтения цвета.

	// Маски для извлечения значений
	maskChar  = (1 << bitsChar) - 1  // 0xFF
	maskColor = (1 << bitsColor) - 1 // 0xFFFFFF
)

// MakeGlyph создает новый Glyph из RGB-цвета и символа.
//
// Параметры:
//   - colorRGB: RGB-цвет в формате 0xRRGGBB (учитываются только младшие 24 бита)
//   - char: ASCII символ для отображения (учитываются младшие 8 бит)
//
// Пример:
//
//	// Оранжевая буква 'A'
//	glyph := MakeGlyph(0xFFA500, 'A')
//	// Внутреннее представление: 0x00FFA541
//	//   - 0x00FFA500 (цвет) << 8 = 0xFFA50000
//	//   - 'A' = 0x41
//	//   - 0xFFA50000 | 0x41 = 0xFFA50041
func MakeGlyph(colorRGB uint32, char byte) Glyph {
	return Glyph((colorRGB&maskColor)<<shiftColor | (uint32(char) & maskChar))
}

// Color извлекает 24-битный RGB-цвет из Glyph.
//
// Возвращает цвет в формате 0xRRGGBB.
// Пример для glyph = MakeGlyph(0xFFA500, 'A'):
//
//	color := glyph.Color() // 0xFFA500
func (g Glyph) Color() uint32 {
	return uint32(g>>shiftColor) & maskColor
}

// Char извлекает символ из Glyph.
//
// Возвращает ASCII символ.
// Пример для glyph = MakeGlyph(0xFFA500, 'A'):
//
//	char := glyph.Char() // 'A' (0x41)
func (g Glyph) Char() byte {
	return byte(g & maskChar)
}

// String возвращает человеко-читаемое представление Glyph.
// Реализует интерфейс fmt.Stringer.
// Формат: "Glyph{char='A', color=#FFA500}"
func (g Glyph) String() string {
	const hex = "0123456789ABCDEF"

	char := g.Char()
	color := g.Color()

	var buf [64]byte
	n := 0

	// "Glyph{char='"
	copy(buf[n:], "Glyph{char='")
	n += len("Glyph{char='")

	// char
	if char >= 32 && char <= 126 {
		buf[n] = char
		n++
	} else {
		buf[n] = '\\'
		buf[n+1] = 'x'
		buf[n+2] = hex[char>>4]
		buf[n+3] = hex[char&0xF]
		n += 4
	}

	// "', color=#"
	copy(buf[n:], "', color=#")
	n += len("', color=#")

	// color hex (RRGGBB)
	buf[n+0] = hex[(color>>20)&0xF]
	buf[n+1] = hex[(color>>16)&0xF]
	buf[n+2] = hex[(color>>12)&0xF]
	buf[n+3] = hex[(color>>8)&0xF]
	buf[n+4] = hex[(color>>4)&0xF]
	buf[n+5] = hex[color&0xF]
	n += 6

	// "}"
	buf[n] = '}'
	n++

	return string(buf[:n])
}

// HexColor возвращает строковое HEX-представление цвета (например, "#00FF00").
func (g Glyph) HexColor() string {
	const hex = "0123456789ABCDEF"

	c := g.Color()

	var buf [7]byte
	buf[0] = '#'
	buf[1] = hex[(c>>20)&0xF]
	buf[2] = hex[(c>>16)&0xF]
	buf[3] = hex[(c>>12)&0xF]
	buf[4] = hex[(c>>8)&0xF]
	buf[5] = hex[(c>>4)&0xF]
	buf[6] = hex[c&0xF]

	return string(buf[:])
}
