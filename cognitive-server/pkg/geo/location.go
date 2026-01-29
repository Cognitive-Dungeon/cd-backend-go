package geo

import "fmt"

// Location — это упакованная 3D координата (X, Y, Z).
// Она занимает 8 байт и передается по значению (без аллокаций).
//
// Layout (64 бита):
// [ Z (12 бит) | Y (26 бит) | X (26 бит) ]
//
// Диапазоны:
// X, Y: +/- 33,554,432
// Z:    +/- 2,048
type Location uint64

const (
	// Битовые размеры
	bitsZ = 12
	bitsY = 26
	bitsX = 26

	// Сдвиги
	shiftX = 0
	shiftY = bitsX         // 26
	shiftZ = bitsX + bitsY // 52

	// Маски
	maskX = (1 << bitsX) - 1 // 0x3FFFFFF
	maskY = (1 << bitsY) - 1 // 0x3FFFFFF
	maskZ = (1 << bitsZ) - 1 // 0xFFF

	// Смещения (Offsets) для хранения отрицательных чисел
	offsetX = 1 << (bitsX - 1) // 33,554,432
	offsetY = 1 << (bitsY - 1) // 33,554,432
	offsetZ = 1 << (bitsZ - 1) // 2,048
)

// Pos создает Location из координат.
func Pos(x, y, z int) Location {
	// 1. Прибавляем offset (переводим в unsigned диапазон)
	// 2. Обрезаем маской (защита от переполнения)
	// 3. Сдвигаем на позицию
	ux := uint64(x+offsetX) & maskX
	uy := uint64(y+offsetY) & maskY
	uz := uint64(z+offsetZ) & maskZ

	return Location((uz << shiftZ) | (uy << shiftY) | ux)
}

// X возвращает координату X.
func (l Location) X() int {
	// Сдвигаем обратно -> маскируем -> отнимаем offset
	val := (uint64(l) >> shiftX) & maskX
	return int(val) - offsetX
}

// Y возвращает координату Y.
func (l Location) Y() int {
	val := (uint64(l) >> shiftY) & maskY
	return int(val) - offsetY
}

// Z возвращает координату Z.
func (l Location) Z() int {
	val := (uint64(l) >> shiftZ) & maskZ
	return int(val) - offsetZ
}

// XYZ возвращает все три координаты сразу (деструктуризация).
func (l Location) XYZ() (x, y, z int) {
	return l.X(), l.Y(), l.Z()
}

// Shift возвращает новую координату, смещенную на dx, dy, dz.
func (l Location) Shift(dx, dy, dz int) Location {
	return Pos(l.X()+dx, l.Y()+dy, l.Z()+dz)
}

// String реализует интерфейс fmt.Stringer для удобного логирования.
func (l Location) String() string {
	return fmt.Sprintf("(%d, %d, %d)", l.X(), l.Y(), l.Z())
}
