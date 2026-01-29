package types

import (
	"fmt"
	"testing"
)

func TestMakeGlyph(t *testing.T) {
	type args struct {
		colorRGB uint32
		char     byte
	}

	tests := []struct {
		name string
		args args
		want Glyph
	}{
		{
			name: "basic - orange A",
			args: args{colorRGB: 0xFFA500, char: 'A'},
			want: Glyph(0xFFA50041), // 0xFFA500 << 8 | 0x41
		},
		{
			name: "black space",
			args: args{colorRGB: 0x000000, char: ' '},
			want: Glyph(0x00000020),
		},
		{
			name: "white newline",
			args: args{colorRGB: 0xFFFFFF, char: '\n'},
			want: Glyph(0xFFFFFF0A),
		},
		{
			name: "red exclamation",
			args: args{colorRGB: 0xFF0000, char: '!'},
			want: Glyph(0xFF000021),
		},
		{
			name: "color truncation (ignore alpha)",
			args: args{colorRGB: 0x12345678, char: 'x'},
			want: Glyph(0x34567878), // Берется только 0x345678 (младшие 24 бита)
		},
		{
			name: "zero char",
			args: args{colorRGB: 0x808080, char: 0},
			want: Glyph(0x80808000),
		},
		{
			name: "max char",
			args: args{colorRGB: 0x404040, char: 0xFF},
			want: Glyph(0x404040FF),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakeGlyph(tt.args.colorRGB, tt.args.char); got != tt.want {
				t.Errorf("MakeGlyph() = 0x%08X, want 0x%08X", got, tt.want)
			}
		})
	}
}

func TestGlyph_Char(t *testing.T) {
	tests := []struct {
		name string
		g    Glyph
		want byte
	}{
		{"orange A", Glyph(0xFFA50041), 'A'},
		{"black space", Glyph(0x00000020), ' '},
		{"white newline", Glyph(0xFFFFFF0A), '\n'},
		{"red exclamation", Glyph(0xFF000021), '!'},
		{"zero char", Glyph(0x80808000), 0},
		{"max char", Glyph(0x404040FF), 0xFF},
		{"char only matters in low 8 bits", Glyph(0x12345678), 0x78},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.Char(); got != tt.want {
				t.Errorf("Char() = 0x%02X (%q), want 0x%02X (%q)",
					got, string(got), tt.want, string(tt.want))
			}
		})
	}
}

func TestGlyph_Color(t *testing.T) {
	tests := []struct {
		name string
		g    Glyph
		want uint32
	}{
		{"orange A", Glyph(0xFFA50041), 0xFFA500},
		{"black space", Glyph(0x00000020), 0x000000},
		{"white newline", Glyph(0xFFFFFF0A), 0xFFFFFF},
		{"red exclamation", Glyph(0xFF000021), 0xFF0000},
		{"green B", Glyph(0x00FF0042), 0x00FF00},
		{"blue C", Glyph(0x0000FF43), 0x0000FF},
		{"color extraction ignores char", Glyph(0x12345678), 0x123456},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.Color(); got != tt.want {
				t.Errorf("Color() = 0x%06X, want 0x%06X", got, tt.want)
			}
		})
	}
}

func TestGlyph_HexColor(t *testing.T) {
	tests := []struct {
		name string
		g    Glyph
		want string
	}{
		{"orange", Glyph(0xFFA50041), "#FFA500"},
		{"black", Glyph(0x00000020), "#000000"},
		{"white", Glyph(0xFFFFFF0A), "#FFFFFF"},
		{"red", Glyph(0xFF000021), "#FF0000"},
		{"green", Glyph(0x00FF0042), "#00FF00"},
		{"blue", Glyph(0x0000FF43), "#0000FF"},
		{"padding zeros", Glyph(0x01020304), "#010203"},
		{"mixed case", Glyph(0xA0B0C044), "#A0B0C0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.HexColor(); got != tt.want {
				t.Errorf("HexColor() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestGlyph_String(t *testing.T) {
	tests := []struct {
		name string
		g    Glyph
		want string
	}{
		{
			name: "printable char",
			g:    MakeGlyph(0xFFA500, 'A'),
			want: "Glyph{char='A', color=#FFA500}",
		},
		{
			name: "space",
			g:    MakeGlyph(0x000000, ' '),
			want: "Glyph{char=' ', color=#000000}",
		},
		{
			name: "newline escape",
			g:    MakeGlyph(0xFFFFFF, '\n'),
			want: "Glyph{char='\\x0A', color=#FFFFFF}",
		},
		{
			name: "tab escape",
			g:    MakeGlyph(0x808080, '\t'),
			want: "Glyph{char='\\x09', color=#808080}",
		},
		{
			name: "null char",
			g:    MakeGlyph(0x123456, 0),
			want: "Glyph{char='\\x00', color=#123456}",
		},
		{
			name: "del char",
			g:    MakeGlyph(0x654321, 0x7F),
			want: "Glyph{char='\\x7F', color=#654321}",
		},
		{
			name: "unicode symbol (euro)",
			g:    MakeGlyph(0xFF00FF, 0x80), // € в extended ASCII
			want: "Glyph{char='\\x80', color=#FF00FF}",
		},
		{
			name: "typical bracket",
			g:    MakeGlyph(0x00AA00, '['),
			want: "Glyph{char='[', color=#00AA00}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.g.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Дополнительные тесты для edge cases
func TestGlyph_EdgeCases(t *testing.T) {
	// Проверка, что биты не "протекают" между полями
	t.Run("no cross-field contamination", func(t *testing.T) {
		// Цвет не должен влиять на символ
		g1 := MakeGlyph(0xFFFFFF, 'A')
		g2 := MakeGlyph(0x000000, 'A')
		if g1.Char() != g2.Char() {
			t.Errorf("Char зависит от цвета: %q != %q", g1.Char(), g2.Char())
		}

		// Символ не должен влиять на цвет
		g3 := MakeGlyph(0x123456, 'A')
		g4 := MakeGlyph(0x123456, 'Z')
		if g3.Color() != g4.Color() {
			t.Errorf("Color зависит от символа: 0x%06X != 0x%06X",
				g3.Color(), g4.Color())
		}
	})

	// Проверка максимальных значений
	t.Run("max values", func(t *testing.T) {
		g := MakeGlyph(0xFFFFFF, 0xFF)
		if g.Color() != 0xFFFFFF {
			t.Errorf("Max color failed: got 0x%06X", g.Color())
		}
		if g.Char() != 0xFF {
			t.Errorf("Max char failed: got 0x%02X", g.Char())
		}
	})

	// Проверка симметрии Make/Color/Char
	t.Run("roundtrip symmetry", func(t *testing.T) {
		originalColor := uint32(0xABCDEF)
		originalChar := byte('X')

		g := MakeGlyph(originalColor, originalChar)

		if g.Color() != originalColor {
			t.Errorf("Color roundtrip failed: 0x%06X != 0x%06X",
				g.Color(), originalColor)
		}

		if g.Char() != originalChar {
			t.Errorf("Char roundtrip failed: %q != %q", g.Char(), originalChar)
		}
	})
}

// TODO: Перенести примеры в отдельный файл и пакет types_examples

// Пример создания Glyph и получения его компонентов.
func ExampleMakeGlyph() {
	// Создание оранжевого символа 'A'
	glyph := MakeGlyph(0xFFA500, 'A')

	fmt.Printf("Символ: %c\n", glyph.Char())
	fmt.Printf("Цвет в HEX: %s\n", glyph.HexColor())
	fmt.Printf("Цвет в десятичном виде: %d\n", glyph.Color())
	fmt.Println(glyph.String())

	// Output:
	// Символ: A
	// Цвет в HEX: #FFA500
	// Цвет в десятичном виде: 16753920
	// Glyph{char='A', color=#FFA500}
}

// Пример работы с непечатаемыми символами.
func ExampleGlyph_nonPrintable() {
	// Символ новой строки с зеленым цветом
	glyph := MakeGlyph(0x00FF00, '\n')

	fmt.Println(glyph.String())
	fmt.Printf("Код символа: 0x%02X\n", glyph.Char())

	// Output:
	// Glyph{char='\x0A', color=#00FF00}
	// Код символа: 0x0A
}

// Пример сериализации и десериализации.
func ExampleGlyph_serialization() {
	original := MakeGlyph(0x123456, 'X')

	// Сериализация в uint32 (например, для хранения в БД)
	serialized := uint32(original)
	fmt.Printf("Сериализованное значение: 0x%08X\n", serialized)

	// Десериализация обратно в Glyph
	restored := Glyph(serialized)

	// Проверка, что значения совпадают
	fmt.Printf("Символ совпадает: %v\n", original.Char() == restored.Char())
	fmt.Printf("Цвет совпадает: %v\n", original.Color() == restored.Color())

	// Output:
	// Сериализованное значение: 0x12345658
	// Символ совпадает: true
	// Цвет совпадает: true
}

// Пример использования Glyph в отрисовке.
func ExampleGlyph_drawing() {
	// Создаем палитру символов для простой графики
	palette := []Glyph{
		MakeGlyph(0xFF0000, '#'), // Красная стена
		MakeGlyph(0x00FF00, '.'), // Зеленый пол
		MakeGlyph(0x0000FF, '@'), // Синий игрок
		MakeGlyph(0xFFFF00, 'G'), // Желтый монстр
	}

	// Имитируем отрисовку простого уровня
	for i, glyph := range palette {
		fmt.Printf("%d: %s\n", i, glyph.String())
	}

	// Output:
	// 0: Glyph{char='#', color=#FF0000}
	// 1: Glyph{char='.', color=#00FF00}
	// 2: Glyph{char='@', color=#0000FF}
	// 3: Glyph{char='G', color=#FFFF00}
}
