package types

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestEntityID_IsNil(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want bool
	}{
		{"Zero is Nil", 0, true},
		{"NilID constant is Nil", NilID, true},
		{"Non-zero is not Nil", 12345, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsNil(); got != tt.want {
				t.Errorf("IsNil() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		id      EntityID
		want    []byte
		wantErr bool
	}{
		{"Simple ID", EntityID(123), []byte(`"123"`), false},
		{"Zero ID", EntityID(0), []byte(`"0"`), false},
		{"Large ID", EntityID(18446744073709551615), []byte(`"18446744073709551615"`), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.id.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("MarshalJSON() got = %s, want %s", string(got), string(tt.want))
			}
		})
	}
}

func TestEntityID_Shard(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want uint8
	}{
		{"Shard 1", EntityID(1 << shiftShard), 1},
		{"Shard 255", EntityID(255 << shiftShard), 255},
		{"Shard 0", EntityID(0), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Shard(); got != tt.want {
				t.Errorf("Shard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_String(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want string
	}{
		{"Basic", EntityID(100), "100"},
		{"Zero", EntityID(0), "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_Time(t *testing.T) {
	expectedMillis := int64(customEpoch + 1000)
	expectedTime := time.UnixMilli(expectedMillis)

	id := EntityID(uint64(1000) << shiftTime)

	t.Run("Extract Time", func(t *testing.T) {
		got := id.Time()

		if got.UnixMilli() != expectedTime.UnixMilli() {
			t.Errorf("Time() = %v (unix: %v), want %v (unix: %v)",
				got, got.UnixMilli(), expectedTime, expectedTime.UnixMilli())
		}
	})
}

func TestEntityID_Type(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want uint8
	}{
		{"Type 5", EntityID(5 << shiftType), 5},
		{"Type 31", EntityID(31 << shiftType), 31},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Type(); got != tt.want {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_Sequence(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want uint16
	}{
		{
			name: "sequence 0",
			id:   EntityID(1000<<shiftTime | 1<<shiftShard | 2<<shiftType | 0),
			want: 0,
		},
		{
			name: "sequence 1",
			id:   EntityID(1000<<shiftTime | 1<<shiftShard | 2<<shiftType | 1),
			want: 1,
		},
		{
			name: "sequence 1023 (max)",
			id:   EntityID(1000<<shiftTime | 1<<shiftShard | 2<<shiftType | maskSeq),
			want: uint16(maskSeq),
		},
		{
			name: "sequence masked correctly",
			id:   EntityID(1000<<shiftTime | 1<<shiftShard | 2<<shiftType | 0x1234),
			want: 0x0234, // Только 10 бит
		},
		{
			name: "only sequence bits matter",
			id:   EntityID(5), // Только sequence биты
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Sequence(); got != tt.want {
				t.Errorf("Sequence() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantID  EntityID
		wantErr bool
	}{
		{"String ID", []byte(`"123"`), EntityID(123), false},
		{"Number ID", []byte(`456`), EntityID(456), false},
		{"Empty string", []byte(`""`), EntityID(0), false},
		{"Invalid format", []byte(`"abc"`), EntityID(0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id EntityID
			err := id.UnmarshalJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if id != tt.wantID {
				t.Errorf("UnmarshalJSON() got = %v, want %v", id, tt.wantID)
			}
		})
	}
}

func TestIDGenerator_NextID(t *testing.T) {
	gen := NewGenerator(10)
	typeID := uint8(5)

	t.Run("Basic Generation", func(t *testing.T) {
		id, err := gen.NextID(typeID)
		if err != nil {
			t.Fatalf("NextID() error = %v", err)
		}
		if id.Shard() != 10 {
			t.Errorf("Shard() = %v, want 10", id.Shard())
		}
		if id.Type() != typeID {
			t.Errorf("Type() = %v, want %v", id.Type(), typeID)
		}
		// Проверяем, что sequence >= 0
		if id.Sequence() < 0 || id.Sequence() > 1023 {
			t.Errorf("Sequence() = %v, want between 0 and 1023", id.Sequence())
		}
	})

	t.Run("Sequence Increment", func(t *testing.T) {
		// Генерируем два ID подряд в одну мс (обычно успевает)
		id1, _ := gen.NextID(typeID)
		id2, _ := gen.NextID(typeID)

		if id1 == id2 {
			t.Errorf("NextID() generated duplicate IDs: %v", id1)
		}

		// Если время то же самое, sequence должен отличаться
		if id1.Time().UnixMilli() == id2.Time().UnixMilli() {
			s1 := id1.Sequence()
			s2 := id2.Sequence()
			if s2 != s1+1 {
				t.Errorf("Sequence should increment, got %v -> %v", s1, s2)
			}
		}
	})

	t.Run("Sequence Wraps Correctly", func(t *testing.T) {
		// Создаем генератор с почти полным sequence
		g := &IDGenerator{
			shardID:  1,
			lastTime: uint64(time.Now().UnixMilli()) - customEpoch,
			sequence: maskSeq - 1, // 1022
		}

		// Первый ID: sequence = 1023
		id1, err := g.NextID(typeID)
		if err != nil {
			t.Fatalf("NextID() error = %v", err)
		}
		if id1.Sequence() != maskSeq {
			t.Errorf("Expected sequence %v, got %v", maskSeq, id1.Sequence())
		}

		// Второй ID: sequence должен переполниться и время увеличиться
		id2, err := g.NextID(typeID)
		if err != nil {
			t.Fatalf("NextID() error = %v", err)
		}
		if id2.Sequence() != 0 {
			t.Errorf("After wrap, expected sequence 0, got %v", id2.Sequence())
		}
		// Время должно увеличиться хотя бы на 1 мс
		if id2.Time().UnixMilli() <= id1.Time().UnixMilli() {
			t.Errorf("After sequence wrap, time should increase")
		}
	})
}

func TestNewGenerator(t *testing.T) {
	shardID := uint8(15)
	got := NewGenerator(shardID)
	if got.shardID != uint64(shardID) {
		t.Errorf("NewGenerator() shardID = %v, want %v", got.shardID, shardID)
	}
}

// Дополнительный тест на конкурентность
func TestGenerator_Concurrency(t *testing.T) {
	gen := NewGenerator(1)
	count := 1000
	var wg sync.WaitGroup
	ids := make(chan EntityID, count)

	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			id, _ := gen.NextID(1)
			ids <- id
		}()
	}

	wg.Wait()
	close(ids)

	unique := make(map[EntityID]bool)
	for id := range ids {
		if unique[id] {
			t.Errorf("Duplicate ID found: %v", id)
		}
		unique[id] = true
	}

	if len(unique) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(unique))
	}
}

// TODO: Перенести примеры в отдельный файл и пакет types_examples

// ExampleNewGenerator показывает, как создать генератор и получить новый ID.
func ExampleNewGenerator() {
	// Создаем генератор для шарда №1
	gen := NewGenerator(1)

	// Генерируем ID для сущности типа №5
	id, err := gen.NextID(5)
	if err != nil {
		return
	}

	fmt.Printf("Shard ID: %d\n", id.Shard())
	fmt.Printf("Type ID: %d\n", id.Type())
	// Output:
	// Shard ID: 1
	// Type ID: 5
}

// ExampleEntityID показывает, как извлекать данные из существующего ID.
func ExampleEntityID() {
	// Допустим, у нас есть готовый ID (число взято для примера)
	// В реальности он будет получен через gen.NextID()
	var id EntityID = 8388641797

	// Можно проверить на Nil
	if !id.IsNil() {
		fmt.Printf("ID string: %s\n", id.String())
		fmt.Printf("Timestamp: %d\n", id.Time().UnixMilli())
		fmt.Printf("Type: %d\n", id.Type())
		fmt.Printf("Shard: %d\n", id.Shard())
		fmt.Printf("Sequence: %d\n", id.Sequence())
	}
	// Output:
	// ID string: 8388641797
	// Timestamp: 1764806401000
	// Type: 1
	// Shard: 1
	// Sequence: 5
}

// ExampleEntityID_MarshalJSON показывает, как ID превращается в строку в JSON.
// Это важно для совместимости с JavaScript (числа > 2^53-1).
func ExampleEntityID_MarshalJSON() {
	type User struct {
		ID   EntityID `json:"id"`
		Name string   `json:"name"`
	}

	u := User{
		ID:   EntityID(123456789),
		Name: "John",
	}

	data, _ := json.Marshal(u)
	fmt.Println(string(data))
	// Output: {"id":"123456789","name":"John"}
}

// ExampleEntityID_UnmarshalJSON показывает, как парсить ID из JSON.
// Поддерживаются и строки, и числа.
func ExampleEntityID_UnmarshalJSON() {
	jsonData := []byte(`{"id": "9876543210"}`)

	var obj struct {
		ID EntityID `json:"id"`
	}

	if err := json.Unmarshal(jsonData, &obj); err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Printf("Parsed ID: %d\n", obj.ID)
	// Output: Parsed ID: 9876543210
}

// ExampleEntityID_Time показывает, как узнать время создания ID.
func ExampleEntityID_Time() {
	// Создадим генератор
	gen := NewGenerator(1)
	id, _ := gen.NextID(1)

	// Получаем время (обрезаем до секунд для стабильности примера)
	t := id.Time().UTC()

	// В реальности здесь будет дата, близкая к time.Now()
	// Мы просто проверим, что год корректный (после 2025)
	if t.Year() >= 2025 {
		fmt.Println("ID year is 2025 or later")
	}
	// Output: ID year is 2025 or later
}
