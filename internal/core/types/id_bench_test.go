package types

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/*
   Sinks — обязательны.
   Нужны, чтобы компилятор не выкинул вычисления.
*/

var (
	sinkID     EntityID
	sinkU8     uint8
	sinkU16    uint16
	sinkU32    uint32
	sinkUint64 uint64
)

/*
   =========================
   noinline helpers
   =========================
*/

//go:noinline
func packEntityIDNoInline(
	shard uint8,
	typ uint8,
	gen uint16,
	index uint32,
) EntityID {
	return PackEntityID(shard, typ, gen, index)
}

//go:noinline
func entityIDShardNoInline(id EntityID) uint8 {
	return id.Shard()
}

//go:noinline
func entityIDTypeNoInline(id EntityID) uint8 {
	return id.Type()
}

//go:noinline
func entityIDGenNoInline(id EntityID) uint16 {
	return id.Generation()
}

//go:noinline
func entityIDIndexNoInline(id EntityID) uint32 {
	return id.Index()
}

/*
   =========================
   Benchmarks: EntityID
   =========================
*/

func BenchmarkPackEntityID(b *testing.B) {
	var id EntityID
	for i := 0; i < b.N; i++ {
		id = packEntityIDNoInline(
			1,
			2,
			uint16(i),
			uint32(i),
		)
	}
	sinkID = id
}

func BenchmarkEntityID_Getters(b *testing.B) {
	id := packEntityIDNoInline(1, 2, 3, 4)

	b.Run("Shard", func(b *testing.B) {
		var v uint8
		for i := 0; i < b.N; i++ {
			v = entityIDShardNoInline(id)
		}
		sinkU8 = v
	})

	b.Run("Type", func(b *testing.B) {
		var v uint8
		for i := 0; i < b.N; i++ {
			v = entityIDTypeNoInline(id)
		}
		sinkU8 = v
	})

	b.Run("Gen", func(b *testing.B) {
		var v uint16
		for i := 0; i < b.N; i++ {
			v = entityIDGenNoInline(id)
		}
		sinkU16 = v
	})

	b.Run("Index", func(b *testing.B) {
		var v uint32
		for i := 0; i < b.N; i++ {
			v = entityIDIndexNoInline(id)
		}
		sinkU32 = v
	})
}

/*
   =========================
   Snowflake benchmarks
   =========================
*/

// Эпоха: 25 декабря 2004 года, 00:00:00 UTC
var snowflakeEpoch = time.Date(2004, 12, 25, 0, 0, 0, 0, time.UTC)

// Constants for bit positions
const (
	snowflakeSeqBits       = 10
	snowflakeTypeBits      = 5
	snowflakeShardBits     = 8
	snowflakeTimestampBits = 41

	snowflakeSeqShift       = 0
	snowflakeTypeShift      = snowflakeSeqShift + snowflakeSeqBits
	snowflakeShardShift     = snowflakeTypeShift + snowflakeTypeBits
	snowflakeTimestampShift = snowflakeShardShift + snowflakeShardBits

	snowflakeSeqMask       = (1 << snowflakeSeqBits) - 1       // 0x3FF
	snowflakeTypeMask      = (1 << snowflakeTypeBits) - 1      // 0x1F
	snowflakeShardMask     = (1 << snowflakeShardBits) - 1     // 0xFF
	snowflakeTimestampMask = (1 << snowflakeTimestampBits) - 1 // 0x1FFFFFFFFFF
)

// SnowflakeGenerator generates snowflake IDs
type SnowflakeGenerator struct {
	lastTimestamp int64
	sequence      uint16
	shardID       uint8
	mutex         sync.Mutex
}

// NewSnowflakeGenerator creates a new snowflake generator
func NewSnowflakeGenerator(shardID uint8) *SnowflakeGenerator {
	if shardID > snowflakeShardMask {
		panic("shardID exceeds maximum value")
	}
	return &SnowflakeGenerator{
		shardID: shardID,
	}
}

// Next generates a new snowflake ID
func (g *SnowflakeGenerator) Next(typeID uint8) uint64 {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	now := time.Since(snowflakeEpoch).Milliseconds()

	if now == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & snowflakeSeqMask
		if g.sequence == 0 {
			// Sequence overflow, wait for next millisecond
			for now <= g.lastTimestamp {
				now = time.Since(snowflakeEpoch).Milliseconds()
			}
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = now

	return PackSnowflakeID(uint64(now), g.shardID, typeID, g.sequence)
}

// PackSnowflakeID packs components into a snowflake ID
func PackSnowflakeID(timestamp uint64, shardID uint8, typeID uint8, sequence uint16) uint64 {
	if timestamp > snowflakeTimestampMask {
		panic("timestamp exceeds 41 bits")
	}
	if shardID > snowflakeShardMask {
		panic("shardID exceeds 8 bits")
	}
	if typeID > snowflakeTypeMask {
		panic("typeID exceeds 5 bits")
	}
	if sequence > snowflakeSeqMask {
		panic("sequence exceeds 10 bits")
	}

	return (timestamp << snowflakeTimestampShift) |
		(uint64(shardID) << snowflakeShardShift) |
		(uint64(typeID) << snowflakeTypeShift) |
		(uint64(sequence) << snowflakeSeqShift)
}

// GetSnowflakeTimestamp extracts timestamp from snowflake ID
func GetSnowflakeTimestamp(id uint64) uint64 {
	return (id >> snowflakeTimestampShift) & snowflakeTimestampMask
}

// GetSnowflakeShardID extracts shard ID from snowflake ID
func GetSnowflakeShardID(id uint64) uint8 {
	return uint8((id >> snowflakeShardShift) & snowflakeShardMask)
}

// GetSnowflakeTypeID extracts type ID from snowflake ID
func GetSnowflakeTypeID(id uint64) uint8 {
	return uint8((id >> snowflakeTypeShift) & snowflakeTypeMask)
}

// GetSnowflakeSequence extracts sequence from snowflake ID
func GetSnowflakeSequence(id uint64) uint16 {
	return uint16((id >> snowflakeSeqShift) & snowflakeSeqMask)
}

// GetSnowflakeTime converts timestamp to time.Time
func GetSnowflakeTime(id uint64) time.Time {
	timestamp := GetSnowflakeTimestamp(id)
	return snowflakeEpoch.Add(time.Duration(timestamp) * time.Millisecond)
}

//go:noinline
func packSnowflakeIDNoInline(timestamp uint64, shardID uint8, typeID uint8, sequence uint16) uint64 {
	return PackSnowflakeID(timestamp, shardID, typeID, sequence)
}

//go:noinline
func snowflakeTimestampNoInline(id uint64) uint64 {
	return GetSnowflakeTimestamp(id)
}

//go:noinline
func snowflakeShardNoInline(id uint64) uint8 {
	return GetSnowflakeShardID(id)
}

//go:noinline
func snowflakeTypeNoInline(id uint64) uint8 {
	return GetSnowflakeTypeID(id)
}

//go:noinline
func snowflakeSequenceNoInline(id uint64) uint16 {
	return GetSnowflakeSequence(id)
}

//go:noinline
func snowflakeNextNoInline(gen *SnowflakeGenerator, typeID uint8) uint64 {
	return gen.Next(typeID)
}

func BenchmarkSnowflakeID(b *testing.B) {
	gen := NewSnowflakeGenerator(1)

	b.Run("Pack", func(b *testing.B) {
		var id uint64
		for i := 0; i < b.N; i++ {
			id = packSnowflakeIDNoInline(
				uint64(i)%snowflakeTimestampMask,
				1,
				2,
				uint16(i%1024),
			)
		}
		sinkUint64 = id
	})

	b.Run("Next", func(b *testing.B) {
		var id uint64
		for i := 0; i < b.N; i++ {
			id = snowflakeNextNoInline(gen, uint8(i%32))
		}
		sinkUint64 = id
	})

	b.Run("Getters", func(b *testing.B) {
		id := packSnowflakeIDNoInline(1234567890, 1, 2, 3)

		b.Run("Timestamp", func(b *testing.B) {
			var v uint64
			for i := 0; i < b.N; i++ {
				v = snowflakeTimestampNoInline(id)
			}
			sinkUint64 = v
		})

		b.Run("Shard", func(b *testing.B) {
			var v uint8
			for i := 0; i < b.N; i++ {
				v = snowflakeShardNoInline(id)
			}
			sinkU8 = v
		})

		b.Run("Type", func(b *testing.B) {
			var v uint8
			for i := 0; i < b.N; i++ {
				v = snowflakeTypeNoInline(id)
			}
			sinkU8 = v
		})

		b.Run("Sequence", func(b *testing.B) {
			var v uint16
			for i := 0; i < b.N; i++ {
				v = snowflakeSequenceNoInline(id)
			}
			sinkU16 = v
		})
	})

	b.Run("TimeConversion", func(b *testing.B) {
		id := packSnowflakeIDNoInline(1234567890, 1, 2, 3)
		var t time.Time

		b.Run("GetTime", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				t = GetSnowflakeTime(id)
			}
			// Используем время чтобы компилятор не оптимизировал
			sinkUint64 = uint64(t.UnixNano())
		})
	})
}

/*
   =========================
   Atomic baseline (sanity)
   =========================
*/

var atomicCounter uint64

//go:noinline
func atomicAddNoInline(delta uint64) uint64 {
	return atomic.AddUint64(&atomicCounter, delta)
}

//go:noinline
func atomicLoadNoInline() uint64 {
	return atomic.LoadUint64(&atomicCounter)
}

func BenchmarkAtomic(b *testing.B) {
	b.Run("Add", func(b *testing.B) {
		var v uint64
		for i := 0; i < b.N; i++ {
			v = atomicAddNoInline(1)
		}
		sinkUint64 = v
	})

	b.Run("Load", func(b *testing.B) {
		var v uint64
		for i := 0; i < b.N; i++ {
			v = atomicLoadNoInline()
		}
		sinkUint64 = v
	})
}

/*
   =========================
   String ID benchmarks
   =========================
*/

//go:noinline
func createStringIDNoInline(base string, suffix int) string {
	return base + "_" + strconv.Itoa(suffix)
}

//go:noinline
func parseStringIDNoInline(id string) (string, string, error) {
	parts := strings.Split(id, "_")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid id format")
	}
	prefix := strings.Join(parts[:len(parts)-1], "_")
	suffix := parts[len(parts)-1]
	return prefix, suffix, nil
}

//go:noinline
func extractStringPartNoInline(id string, index int) string {
	parts := strings.Split(id, "_")
	if index < 0 || index >= len(parts) {
		return ""
	}
	return parts[index]
}

//go:noinline
func compareStringIDNoInline(id1, id2 string) bool {
	return id1 == id2
}

var sinkString string
var sinkBool bool

func BenchmarkStringID(b *testing.B) {
	b.Run("CreateSimple", func(b *testing.B) {
		var s string
		for i := 0; i < b.N; i++ {
			s = createStringIDNoInline("stair_exit_to_level", i%100)
		}
		sinkString = s
	})

	b.Run("CreateFormatted", func(b *testing.B) {
		var s string
		for i := 0; i < b.N; i++ {
			s = fmt.Sprintf("shard_%d_type_%d_gen_%d_idx_%d",
				uint8(i),
				uint8(i>>8),
				uint16(i>>16),
				uint32(i),
			)
		}
		sinkString = s
	})

	b.Run("ParseComponents", func(b *testing.B) {
		id := "shard_1_type_2_gen_3_idx_4"
		var prefix, suffix string
		var err error

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			prefix, suffix, err = parseStringIDNoInline(id)
		}

		sinkString = prefix + suffix
		_ = err
	})

	b.Run("ExtractPart", func(b *testing.B) {
		id := "stair_exit_to_level_1"
		var part string

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			part = extractStringPartNoInline(id, i%4)
		}
		sinkString = part
	})

	b.Run("Compare", func(b *testing.B) {
		id1 := "stair_exit_to_level_1"
		id2 := "stair_exit_to_level_2"
		var equal bool

		b.Run("Equal", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				equal = compareStringIDNoInline(id1, id1)
			}
			sinkBool = equal
		})

		b.Run("NotEqual", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				equal = compareStringIDNoInline(id1, id2)
			}
			sinkBool = equal
		})
	})

	b.Run("MapLookup", func(b *testing.B) {
		// Подготовка данных для теста поиска в map
		m := make(map[string]int)
		for i := 0; i < 1000; i++ {
			m[fmt.Sprintf("entity_%d", i)] = i
		}

		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}

		var value int
		var found bool

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			key := keys[i%len(keys)]
			value, found = m[key]
		}

		sinkUint64 = uint64(value)
		sinkBool = found
	})
}
