package types

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
)

func TestObjectGuid_Generation(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectGuid
		want uint16
	}{
		{
			name: "Generation zero",
			id:   ObjectGuid(0),
			want: 0,
		},
		{
			name: "Generation simple",
			id:   ObjectGuid(uint64(1) << shiftGen),
			want: 1,
		},
		{
			name: "Generation max",
			id:   ObjectGuid(uint64(maskGen) << shiftGen),
			want: maskGen,
		},
		{
			name: "Generation masked correctly",
			id:   ObjectGuid(uint64(0xFFFFFFFF) << shiftGen),
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

func TestObjectGuid_Index(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectGuid
		want uint32
	}{
		{
			name: "Index zero",
			id:   ObjectGuid(0),
			want: 0,
		},
		{
			name: "Index simple",
			id:   ObjectGuid(42),
			want: 42,
		},
		{
			name: "Index max",
			id:   ObjectGuid(maskIndex),
			want: maskIndex,
		},
		{
			name: "Index masked correctly",
			id:   ObjectGuid(uint64(maskIndex) | (1 << shiftGen)),
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

func TestObjectGuid_IsLocal(t *testing.T) {
	id := PackObjectGuid(5, 1, 0, 10)

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

func TestNilObjectGuidPartsAreZero(t *testing.T) {
	var id ObjectGuid
	if id.Shard() != 0 || id.Type() != 0 || id.Generation() != 0 || id.Index() != 0 {
		t.Fatal("NilObjectGuid must have all parts equal to zero")
	}
}

func TestObjectGuid_IsNil(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectGuid
		want bool
	}{
		{"Zero is Nil", 0, true},
		{"NilObjectGuid constant", NilObjectGuid, true},
		{"Non-zero is not Nil", PackObjectGuid(1, 1, 1, 1), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.IsNil(); got != tt.want {
				t.Errorf("IsNil() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPackObjectGuid_Masking(t *testing.T) {
	id := PackObjectGuid(255, 255, 65535, 0xFFFFFFFF)

	if id.Shard() != maskShard ||
		id.Type() != maskType ||
		id.Generation() != maskGen ||
		id.Index() != maskIndex {
		t.Fatal("PackObjectGuid masking failed")
	}
}

func TestObjectGuid_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectGuid
		want []byte
	}{
		{
			name: "Simple ID",
			id:   PackObjectGuid(1, 2, 3, 4),
			want: []byte(`"` + strconv.FormatUint(uint64(PackObjectGuid(1, 2, 3, 4)), 10) + `"`),
		},
		{
			name: "Zero ID",
			id:   ObjectGuid(0),
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

func TestObjectGuid_Shard(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectGuid
		want uint8
	}{
		{
			name: "Shard zero",
			id:   ObjectGuid(0),
			want: 0,
		},
		{
			name: "Shard simple",
			id:   ObjectGuid(uint64(5) << shiftShard),
			want: 5,
		},
		{
			name: "Shard max",
			id:   ObjectGuid(uint64(maskShard) << shiftShard),
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

func TestObjectGuid_String(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectGuid
	}{
		{"Nil", ObjectGuid(0)},
		{"Non-nil", PackObjectGuid(1, 2, 3, 4)},
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

func TestObjectGuid_Type(t *testing.T) {
	tests := []struct {
		name string
		id   ObjectGuid
		want uint8
	}{
		{
			name: "Type zero",
			id:   ObjectGuid(0),
			want: 0,
		},
		{
			name: "Type simple",
			id:   ObjectGuid(uint64(7) << shiftType),
			want: 7,
		},
		{
			name: "Type max",
			id:   ObjectGuid(uint64(maskType) << shiftType),
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

func TestObjectGuid_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    ObjectGuid
		wantErr bool
	}{
		{
			name: "String ID",
			data: []byte(`"123"`),
			want: ObjectGuid(123),
		},
		{
			name: "Number ID",
			data: []byte(`456`),
			want: ObjectGuid(456),
		},
		{
			name: "Empty string",
			data: []byte(`""`),
			want: ObjectGuid(0),
		},
		{
			name:    "Invalid format",
			data:    []byte(`"abc"`),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id ObjectGuid
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

func TestPackObjectGuid(t *testing.T) {
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
			id := PackObjectGuid(tt.shard, tt.typ, tt.gen, tt.index)

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

func TestObjectGuid_JSONRoundTrip(t *testing.T) {
	original := PackObjectGuid(3, 4, 5, 6)

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded ObjectGuid
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if decoded != original {
		t.Errorf("JSON round-trip failed: got %v, want %v", decoded, original)
	}
}

// FuzzPackObjectGuid проверяет инвариант:
// PackObjectGuid → извлечение полей → равенство исходным значениям.
func FuzzPackObjectGuid(f *testing.F) {
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
		id := PackObjectGuid(shard, typ, gen, index)

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

func FuzzObjectGuid_JSONRoundTrip(f *testing.F) {
	f.Add(uint64(0))
	f.Add(uint64(1))
	f.Add(uint64(123456789))
	f.Add(^uint64(0)) // max uint64

	f.Fuzz(func(t *testing.T, raw uint64) {
		original := ObjectGuid(raw)

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Marshal failed: %v", err)
		}

		var decoded ObjectGuid
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

func FuzzObjectGuid_UnmarshalJSON(f *testing.F) {
	f.Add([]byte(`"123"`))
	f.Add([]byte(`123`))
	f.Add([]byte(`""`))
	f.Add([]byte(`"not-a-number"`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`[]`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var id ObjectGuid
		_ = id.UnmarshalJSON(data)
		// Единственное требование: отсутствие panic
	})
}
