package fuzz

import (
	"encoding/binary"
	"math"
	"time"
)

// Bool extracts a boolean value from the Consumer.
func (c *Consumer) Bool() bool {
	return c.takeByte()%2 == 1
}

// Int extracts an int value from the Consumer (8 bytes).
func (c *Consumer) Int() int {
	return int(c.Int64())
}

// Int8 extracts an int8 value from the Consumer.
func (c *Consumer) Int8() int8 {
	return int8(c.takeByte())
}

// Int16 extracts an int16 value from the Consumer (2 bytes, big-endian).
func (c *Consumer) Int16() int16 {
	return int16(binary.BigEndian.Uint16(c.take(2)))
}

// Int32 extracts an int32 value from the Consumer (4 bytes, big-endian).
func (c *Consumer) Int32() int32 {
	return int32(binary.BigEndian.Uint32(c.take(4)))
}

// Int64 extracts an int64 value from the Consumer (8 bytes, big-endian).
func (c *Consumer) Int64() int64 {
	return int64(binary.BigEndian.Uint64(c.take(8)))
}

// Uint extracts a uint value from the Consumer (8 bytes).
func (c *Consumer) Uint() uint {
	return uint(c.Uint64())
}

// Uint8 extracts a uint8 value from the Consumer.
func (c *Consumer) Uint8() uint8 {
	return c.takeByte()
}

// Uint16 extracts a uint16 value from the Consumer (2 bytes, big-endian).
func (c *Consumer) Uint16() uint16 {
	return binary.BigEndian.Uint16(c.take(2))
}

// Uint32 extracts a uint32 value from the Consumer (4 bytes, big-endian).
func (c *Consumer) Uint32() uint32 {
	return binary.BigEndian.Uint32(c.take(4))
}

// Uint64 extracts a uint64 value from the Consumer (8 bytes, big-endian).
func (c *Consumer) Uint64() uint64 {
	return binary.BigEndian.Uint64(c.take(8))
}

// Float32 extracts a float32 value from the Consumer (4 bytes).
func (c *Consumer) Float32() float32 {
	bits := binary.BigEndian.Uint32(c.take(4))
	return math.Float32frombits(bits)
}

// Float64 extracts a float64 value from the Consumer (8 bytes).
func (c *Consumer) Float64() float64 {
	bits := binary.BigEndian.Uint64(c.take(8))
	return math.Float64frombits(bits)
}

// String extracts a length-prefixed string from the Consumer.
// Uses a uint16 for the length, limiting strings to 65535 bytes.
func (c *Consumer) String() string {
	length := c.Uint16()
	return string(c.take(int(length)))
}

// Time extracts a time.Time value from the Consumer as Unix nanoseconds.
func (c *Consumer) Time() time.Time {
	nsec := c.Int64()
	return time.Unix(0, nsec)
}

// Duration extracts a time.Duration value from the Consumer as nanoseconds.
func (c *Consumer) Duration() time.Duration {
	return time.Duration(c.Int64())
}

// Bytes extracts a variable-length byte slice from the Consumer.
// Uses a uint16 for the length prefix, limiting slices to 65535 bytes.
func (c *Consumer) Bytes() []byte {
	return c.take(int(c.Uint16()))
}

// BytesN extracts exactly n bytes from the Consumer.
func (c *Consumer) BytesN(n int) []byte {
	return c.take(n)
}
