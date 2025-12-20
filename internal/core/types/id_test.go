package types

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestEntityID_Generation(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want uint16
	}{
		{
			name: "Generation zero",
			id:   EntityID(0),
			want: 0,
		},
		{
			name: "Generation simple",
			id:   EntityID(uint64(1) << shiftGen),
			want: 1,
		},
		{
			name: "Generation max",
			id:   EntityID(uint64(maskGen) << shiftGen),
			want: maskGen,
		},
		{
			name: "Generation masked correctly",
			id:   EntityID(uint64(0xFFFFFFFF) << shiftGen),
			want: maskGen,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Generation(); got != tt.want {
				t.Errorf("Generation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_Index(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want uint32
	}{
		{
			name: "Index zero",
			id:   EntityID(0),
			want: 0,
		},
		{
			name: "Index simple",
			id:   EntityID(42),
			want: 42,
		},
		{
			name: "Index max",
			id:   EntityID(maskIndex),
			want: maskIndex,
		},
		{
			name: "Index masked correctly",
			id:   EntityID(uint64(maskIndex) | (1 << shiftGen)),
			want: maskIndex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Index(); got != tt.want {
				t.Errorf("Index() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_IsLocal(t *testing.T) {
	id := PackEntityID(5, 1, 0, 10)

	tests := []struct {
		name         string
		currentShard uint8
		want         bool
	}{
		{"Same shard", 5, true},
		{"Different shard", 4, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := id.IsLocal(tt.currentShard); got != tt.want {
				t.Errorf("IsLocal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_IsNil(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want bool
	}{
		{"Zero is Nil", 0, true},
		{"NilEntityID constant", NilEntityID, true},
		{"Non-zero is not Nil", PackEntityID(1, 1, 1, 1), false},
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
		name string
		id   EntityID
		want []byte
	}{
		{
			name: "Simple ID",
			id:   PackEntityID(1, 2, 3, 4),
			want: []byte(`"` + string("72620556876251140") + `"`),
		},
		{
			name: "Zero ID",
			id:   EntityID(0),
			want: []byte(`"0"`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.id.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
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
		{
			name: "Shard zero",
			id:   EntityID(0),
			want: 0,
		},
		{
			name: "Shard simple",
			id:   EntityID(uint64(5) << shiftShard),
			want: 5,
		},
		{
			name: "Shard max",
			id:   EntityID(uint64(maskShard) << shiftShard),
			want: maskShard,
		},
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
	}{
		{"Nil", EntityID(0)},
		{"Non-nil", PackEntityID(1, 2, 3, 4)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.id.String()
			if s == "" {
				t.Errorf("String() returned empty string")
			}
		})
	}
}

func TestEntityID_Type(t *testing.T) {
	tests := []struct {
		name string
		id   EntityID
		want uint8
	}{
		{
			name: "Type zero",
			id:   EntityID(0),
			want: 0,
		},
		{
			name: "Type simple",
			id:   EntityID(uint64(7) << shiftType),
			want: 7,
		},
		{
			name: "Type max",
			id:   EntityID(uint64(maskType) << shiftType),
			want: maskType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Type(); got != tt.want {
				t.Errorf("Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityID_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    EntityID
		wantErr bool
	}{
		{
			name: "String ID",
			data: []byte(`"123"`),
			want: EntityID(123),
		},
		{
			name: "Number ID",
			data: []byte(`456`),
			want: EntityID(456),
		},
		{
			name: "Empty string",
			data: []byte(`""`),
			want: EntityID(0),
		},
		{
			name:    "Invalid format",
			data:    []byte(`"abc"`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id EntityID
			err := id.UnmarshalJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && id != tt.want {
				t.Errorf("UnmarshalJSON() = %v, want %v", id, tt.want)
			}
		})
	}
}

func TestPackEntityID(t *testing.T) {
	tests := []struct {
		name  string
		shard uint8
		typ   uint8
		gen   uint16
		index uint32
	}{
		{"All zero", 0, 0, 0, 0},
		{"Simple values", 1, 2, 3, 4},
		{"Max values", maskShard, maskType, maskGen, maskIndex},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := PackEntityID(tt.shard, tt.typ, tt.gen, tt.index)

			if id.Shard() != tt.shard {
				t.Errorf("Shard() = %v, want %v", id.Shard(), tt.shard)
			}
			if id.Type() != tt.typ {
				t.Errorf("Type() = %v, want %v", id.Type(), tt.typ)
			}
			if id.Generation() != tt.gen {
				t.Errorf("Generation() = %v, want %v", id.Generation(), tt.gen)
			}
			if id.Index() != tt.index {
				t.Errorf("Index() = %v, want %v", id.Index(), tt.index)
			}
		})
	}
}

func TestEntityID_JSONRoundTrip(t *testing.T) {
	original := PackEntityID(3, 4, 5, 6)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded EntityID
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("JSON round-trip failed: got %v, want %v", decoded, original)
	}
}

// FuzzPackEntityID проверяет инвариант:
// PackEntityID → извлечение полей → равенство исходным значениям.
func FuzzPackEntityID(f *testing.F) {
	// Сидовые значения (важно для воспроизводимости)
	f.Add(uint8(0), uint8(0), uint16(0), uint32(0))
	f.Add(uint8(1), uint8(2), uint16(3), uint32(4))
	f.Add(uint8(255), uint8(255), uint16(65535), uint32(4294967295))

	f.Fuzz(func(
		t *testing.T,
		shard uint8,
		typ uint8,
		gen uint16,
		index uint32,
	) {
		id := PackEntityID(shard, typ, gen, index)

		if got := id.Shard(); got != shard {
			t.Fatalf("Shard mismatch: got %d, want %d", got, shard)
		}
		if got := id.Type(); got != typ {
			t.Fatalf("Type mismatch: got %d, want %d", got, typ)
		}
		if got := id.Generation(); got != gen {
			t.Fatalf("Generation mismatch: got %d, want %d", got, gen)
		}
		if got := id.Index(); got != index {
			t.Fatalf("Index mismatch: got %d, want %d", got, index)
		}
	})
}

func FuzzEntityID_JSONRoundTrip(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(123456789))
	f.Add(^uint64(0)) // max uint64

	f.Fuzz(func(t *testing.T, raw uint64) {
		original := EntityID(raw)

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var decoded EntityID
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal failed: %v", err)
		}

		if decoded != original {
			t.Fatalf(
				"JSON round-trip mismatch: got %d, want %d",
				decoded, original,
			)
		}
	})
}

func FuzzEntityID_UnmarshalJSON(f *testing.F) {
	f.Add([]byte(`"123"`))
	f.Add([]byte(`123`))
	f.Add([]byte(`""`))
	f.Add([]byte(`"not-a-number"`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var id EntityID
		_ = id.UnmarshalJSON(data)
		// Единственное требование: отсутствие panic
	})
}
