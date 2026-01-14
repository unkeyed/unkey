package fuzz_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
)

func TestGet_Bool(t *testing.T) {
	// Even byte -> false, odd byte -> true
	c := fuzz.New(t, []byte{0, 1, 2, 3})

	require.False(t, c.Bool())
	require.True(t, c.Bool())
	require.False(t, c.Bool())
	require.True(t, c.Bool())
}

func TestGet_Integers(t *testing.T) {
	// uint8: single byte
	c := fuzz.New(t, []byte{42})
	require.Equal(t, uint8(42), c.Uint8())

	// int8: single byte
	c = fuzz.New(t, []byte{0xFF})
	require.Equal(t, int8(-1), c.Int8())

	// uint16: 2 bytes big-endian
	c = fuzz.New(t, []byte{0x01, 0x02})
	require.Equal(t, uint16(0x0102), c.Uint16())

	// int16: 2 bytes big-endian
	c = fuzz.New(t, []byte{0xFF, 0xFE})
	require.Equal(t, int16(-2), c.Int16())

	// uint32: 4 bytes big-endian
	c = fuzz.New(t, []byte{0x00, 0x00, 0x00, 0x2A})
	require.Equal(t, uint32(42), c.Uint32())

	// int32: 4 bytes big-endian
	c = fuzz.New(t, []byte{0x00, 0x00, 0x00, 0x2A})
	require.Equal(t, int32(42), c.Int32())

	// uint64: 8 bytes big-endian
	c = fuzz.New(t, []byte{0, 0, 0, 0, 0, 0, 0, 100})
	require.Equal(t, uint64(100), c.Uint64())

	// int64: 8 bytes big-endian
	c = fuzz.New(t, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF})
	require.Equal(t, int64(-1), c.Int64())

	// int: 8 bytes big-endian (same as int64)
	c = fuzz.New(t, []byte{0, 0, 0, 0, 0, 0, 0, 42})
	require.Equal(t, int(42), c.Int())

	// uint: 8 bytes big-endian (same as uint64)
	c = fuzz.New(t, []byte{0, 0, 0, 0, 0, 0, 0, 99})
	require.Equal(t, uint(99), c.Uint())
}

func TestGet_Floats(t *testing.T) {
	// float32: 4 bytes
	c := fuzz.New(t, []byte{0x40, 0x48, 0xF5, 0xC3}) // 3.14 in IEEE 754
	f32 := c.Float32()
	require.InDelta(t, 3.14, f32, 0.001)

	// float64: 8 bytes
	c = fuzz.New(t, []byte{0x40, 0x09, 0x21, 0xFB, 0x54, 0x44, 0x2D, 0x18}) // pi
	f64 := c.Float64()
	require.InDelta(t, 3.14159265358979, f64, 0.0000001)
}

func TestGet_String(t *testing.T) {
	// String: uint16 length prefix + bytes
	// Length = 5, content = "hello"
	data := []byte{0x00, 0x05, 'h', 'e', 'l', 'l', 'o'}
	c := fuzz.New(t, data)

	s := c.String()
	require.Equal(t, "hello", s)
}

func TestGet_Time(t *testing.T) {
	// time.Time: int64 nanoseconds since epoch
	// 1000000000 ns = 1 second
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x3B, 0x9A, 0xCA, 0x00}
	c := fuzz.New(t, data)

	tm := c.Time()
	require.Equal(t, time.Unix(0, 1000000000), tm)
}

func TestGet_Duration(t *testing.T) {
	// time.Duration: int64 nanoseconds
	data := []byte{0x00, 0x00, 0x00, 0x00, 0x3B, 0x9A, 0xCA, 0x00}
	c := fuzz.New(t, data)

	d := c.Duration()
	require.Equal(t, time.Second, d)
}

func TestBytes(t *testing.T) {
	data := []byte{0x00, 0x03, 0xDE, 0xAD, 0xBE}
	c := fuzz.New(t, data)

	b := c.Bytes()
	require.Equal(t, []byte{0xDE, 0xAD, 0xBE}, b)
}

func TestBytesN(t *testing.T) {
	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	c := fuzz.New(t, data)

	b := c.BytesN(3)
	require.Equal(t, []byte{0xDE, 0xAD, 0xBE}, b)

	// Should have 1 byte remaining
	require.Equal(t, 1, c.Remaining())
}

func TestSlice(t *testing.T) {
	// Slice: uint8 length + elements
	// Length = 3, three uint8 values
	data := []byte{0x03, 10, 20, 30}
	c := fuzz.New(t, data)

	s := fuzz.Slice[uint8](c)
	require.Equal(t, []uint8{10, 20, 30}, s)
}

func TestSlice_Empty(t *testing.T) {
	data := []byte{0x00} // length = 0
	c := fuzz.New(t, data)

	s := fuzz.Slice[int](c)
	require.Empty(t, s)
}

func TestSlice_AllTypes(t *testing.T) {
	t.Run("bool", func(t *testing.T) {
		data := []byte{0x02, 0x01, 0x00} // length=2, true, false
		c := fuzz.New(t, data)
		s := fuzz.Slice[bool](c)
		require.Equal(t, []bool{true, false}, s)
	})

	t.Run("int", func(t *testing.T) {
		data := []byte{
			0x01,                                           // length=1
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2A, // 42
		}
		c := fuzz.New(t, data)
		s := fuzz.Slice[int](c)
		require.Equal(t, []int{42}, s)
	})

	t.Run("int8", func(t *testing.T) {
		data := []byte{0x02, 0x01, 0xFF} // length=2, 1, -1
		c := fuzz.New(t, data)
		s := fuzz.Slice[int8](c)
		require.Equal(t, []int8{1, -1}, s)
	})

	t.Run("int16", func(t *testing.T) {
		data := []byte{0x01, 0x00, 0x64} // length=1, 100
		c := fuzz.New(t, data)
		s := fuzz.Slice[int16](c)
		require.Equal(t, []int16{100}, s)
	})

	t.Run("int32", func(t *testing.T) {
		data := []byte{0x01, 0x00, 0x00, 0x00, 0x64} // length=1, 100
		c := fuzz.New(t, data)
		s := fuzz.Slice[int32](c)
		require.Equal(t, []int32{100}, s)
	})

	t.Run("int64", func(t *testing.T) {
		data := []byte{
			0x01,                                           // length=1
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, // 100
		}
		c := fuzz.New(t, data)
		s := fuzz.Slice[int64](c)
		require.Equal(t, []int64{100}, s)
	})

	t.Run("uint", func(t *testing.T) {
		data := []byte{
			0x01,                                           // length=1
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x2A, // 42
		}
		c := fuzz.New(t, data)
		s := fuzz.Slice[uint](c)
		require.Equal(t, []uint{42}, s)
	})

	t.Run("uint16", func(t *testing.T) {
		data := []byte{0x01, 0x00, 0x64} // length=1, 100
		c := fuzz.New(t, data)
		s := fuzz.Slice[uint16](c)
		require.Equal(t, []uint16{100}, s)
	})

	t.Run("uint32", func(t *testing.T) {
		data := []byte{0x01, 0x00, 0x00, 0x00, 0x64} // length=1, 100
		c := fuzz.New(t, data)
		s := fuzz.Slice[uint32](c)
		require.Equal(t, []uint32{100}, s)
	})

	t.Run("uint64", func(t *testing.T) {
		data := []byte{
			0x01,                                           // length=1
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, // 100
		}
		c := fuzz.New(t, data)
		s := fuzz.Slice[uint64](c)
		require.Equal(t, []uint64{100}, s)
	})

	t.Run("float32", func(t *testing.T) {
		data := []byte{0x01, 0x3F, 0x80, 0x00, 0x00} // length=1, 1.0
		c := fuzz.New(t, data)
		s := fuzz.Slice[float32](c)
		require.Equal(t, []float32{1.0}, s)
	})

	t.Run("float64", func(t *testing.T) {
		data := []byte{
			0x01,                                           // length=1
			0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 2.0
		}
		c := fuzz.New(t, data)
		s := fuzz.Slice[float64](c)
		require.Equal(t, []float64{2.0}, s)
	})

	t.Run("string", func(t *testing.T) {
		data := []byte{
			0x01,                 // length=1
			0x00, 0x02, 'h', 'i', // string "hi"
		}
		c := fuzz.New(t, data)
		s := fuzz.Slice[string](c)
		require.Equal(t, []string{"hi"}, s)
	})
}

func TestStruct_Simple(t *testing.T) {
	type Simple struct {
		Flag  bool
		Count uint8
		Name  string
	}

	// Flag: 1 byte (odd = true)
	// Count: 1 byte
	// Name: 2 byte length + content
	data := []byte{
		0x01,       // Flag = true
		0x2A,       // Count = 42
		0x00, 0x03, // Name length = 3
		'f', 'o', 'o', // Name = "foo"
	}
	c := fuzz.New(t, data)

	s := fuzz.Struct[Simple](c)
	require.True(t, s.Flag)
	require.Equal(t, uint8(42), s.Count)
	require.Equal(t, "foo", s.Name)
}

func TestStruct_Nested(t *testing.T) {
	type Inner struct {
		Value int32
	}
	type Outer struct {
		Name  string
		Inner Inner
	}

	data := []byte{
		0x00, 0x03, // Name length = 3
		'b', 'a', 'r', // Name = "bar"
		0x00, 0x00, 0x00, 0x64, // Inner.Value = 100
	}
	c := fuzz.New(t, data)

	s := fuzz.Struct[Outer](c)
	require.Equal(t, "bar", s.Name)
	require.Equal(t, int32(100), s.Inner.Value)
}

func TestStruct_WithSlice(t *testing.T) {
	type WithSlice struct {
		Tags []uint8
	}

	data := []byte{
		0x02,   // Tags length = 2
		10, 20, // Tags = [10, 20]
	}
	c := fuzz.New(t, data)

	s := fuzz.Struct[WithSlice](c)
	require.Equal(t, []uint8{10, 20}, s.Tags)
}

func TestStruct_UnexportedFieldsSkipped(t *testing.T) {
	type WithUnexported struct {
		Public  uint8
		private uint8 //nolint:unused
	}

	data := []byte{0x2A} // Only need 1 byte for Public
	c := fuzz.New(t, data)

	s := fuzz.Struct[WithUnexported](c)
	require.Equal(t, uint8(42), s.Public)
}

func TestStruct_TimeFields(t *testing.T) {
	type WithTime struct {
		Created time.Time
		TTL     time.Duration
	}

	data := []byte{
		0x00, 0x00, 0x00, 0x00, 0x3B, 0x9A, 0xCA, 0x00, // Created: 1 second in ns
		0x00, 0x00, 0x00, 0x00, 0x77, 0x35, 0x94, 0x00, // TTL: 2 seconds in ns
	}
	c := fuzz.New(t, data)

	s := fuzz.Struct[WithTime](c)
	require.Equal(t, time.Unix(0, 1000000000), s.Created)
	require.Equal(t, 2*time.Second, s.TTL)
}

func TestConsumer_Remaining(t *testing.T) {
	c := fuzz.New(t, []byte{1, 2, 3, 4, 5})

	require.Equal(t, 5, c.Remaining())
	require.False(t, c.Exhausted())

	c.Uint8()
	require.Equal(t, 4, c.Remaining())

	c.Uint8()
	c.Uint8()
	c.Uint8()
	c.Uint8()

	require.Equal(t, 0, c.Remaining())
	require.True(t, c.Exhausted())
}

func TestMultipleExtractions(t *testing.T) {
	data := []byte{
		0x01,                   // bool = true
		0x00, 0x00, 0x00, 0x2A, // int32 = 42
		0x00, 0x02, 'h', 'i', // string = "hi"
	}
	c := fuzz.New(t, data)

	b := c.Bool()
	i := c.Int32()
	s := c.String()

	require.True(t, b)
	require.Equal(t, int32(42), i)
	require.Equal(t, "hi", s)
	require.True(t, c.Exhausted())
}

func TestSkipOnExhaustion(t *testing.T) {
	// This test verifies that extraction from an exhausted consumer
	// causes t.Skip() to be called. We run it as a subtest so we can
	// check that it was skipped.
	t.Run("skipped", func(t *testing.T) {
		c := fuzz.New(t, []byte{}) // Empty input
		c.Int32()                  // Needs 4 bytes, has 0

		// If we reach here, skip wasn't called
		t.Fatal("should have been skipped")
	})
}

func TestSkipOnPartialExhaustion(t *testing.T) {
	t.Run("skipped", func(t *testing.T) {
		c := fuzz.New(t, []byte{0x01, 0x02}) // Only 2 bytes
		c.Uint8()                            // OK, consumes 1
		c.Int32()                            // Needs 4, only 1 left

		t.Fatal("should have been skipped")
	})
}

func TestGet_UnsupportedTypePanics(t *testing.T) {
	c := fuzz.New(t, []byte{0x01, 0x02, 0x03, 0x04})

	require.Panics(t, func() {
		fuzz.Slice[complex64](c)
	})
}

func TestStruct_AllIntegerTypes(t *testing.T) {
	type AllInts struct {
		I   int
		I8  int8
		I16 int16
		I32 int32
		I64 int64
		U   uint
		U8  uint8
		U16 uint16
		U32 uint32
		U64 uint64
	}

	data := []byte{
		// int: 8 bytes
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		// int8: 1 byte
		0x02,
		// int16: 2 bytes
		0x00, 0x03,
		// int32: 4 bytes
		0x00, 0x00, 0x00, 0x04,
		// int64: 8 bytes
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05,
		// uint: 8 bytes
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06,
		// uint8: 1 byte
		0x07,
		// uint16: 2 bytes
		0x00, 0x08,
		// uint32: 4 bytes
		0x00, 0x00, 0x00, 0x09,
		// uint64: 8 bytes
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0A,
	}
	c := fuzz.New(t, data)

	s := fuzz.Struct[AllInts](c)
	require.Equal(t, int(1), s.I)
	require.Equal(t, int8(2), s.I8)
	require.Equal(t, int16(3), s.I16)
	require.Equal(t, int32(4), s.I32)
	require.Equal(t, int64(5), s.I64)
	require.Equal(t, uint(6), s.U)
	require.Equal(t, uint8(7), s.U8)
	require.Equal(t, uint16(8), s.U16)
	require.Equal(t, uint32(9), s.U32)
	require.Equal(t, uint64(10), s.U64)
}

func TestStruct_FloatTypes(t *testing.T) {
	type Floats struct {
		F32 float32
		F64 float64
	}

	data := []byte{
		// float32: 4 bytes (IEEE 754 for 1.0)
		0x3F, 0x80, 0x00, 0x00,
		// float64: 8 bytes (IEEE 754 for 2.0)
		0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	c := fuzz.New(t, data)

	s := fuzz.Struct[Floats](c)
	require.Equal(t, float32(1.0), s.F32)
	require.Equal(t, float64(2.0), s.F64)
}

func TestStruct_UnsupportedFieldPanics(t *testing.T) {
	type WithChannel struct {
		Ch chan int
	}

	c := fuzz.New(t, []byte{0x01, 0x02, 0x03, 0x04})

	require.Panics(t, func() {
		fuzz.Struct[WithChannel](c)
	})
}

func TestSkipOnTakeByteExhaustion(t *testing.T) {
	// Test the takeByte path specifically when already exhausted
	t.Run("skipped", func(t *testing.T) {
		c := fuzz.New(t, []byte{0x01})
		c.Uint8() // Consume the only byte
		c.Bool()  // takeByte on empty

		t.Fatal("should have been skipped")
	})
}

func FuzzSeed(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		// Just verify we can create a consumer from seeded data
		c := fuzz.New(t, data)
		_ = c.Remaining()
	})
}
