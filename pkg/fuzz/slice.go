package fuzz

// Slice extracts a variable-length slice of type T from the Consumer.
//
// The length is determined by consuming a uint8 from the input (max 255 elements),
// then extracting that many values of type T. Skips if insufficient bytes remain.
//
// Supported element types are the same as the Consumer methods: bool, int, int8,
// int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64,
// string, time.Time, and time.Duration.
func Slice[T any](c *Consumer) []T {
	length := int(c.Uint8())
	result := make([]T, length)
	for i := range length {
		var zero T
		switch any(zero).(type) {
		case bool:
			result[i] = any(c.Bool()).(T)
		case int:
			result[i] = any(c.Int()).(T)
		case int8:
			result[i] = any(c.Int8()).(T)
		case int16:
			result[i] = any(c.Int16()).(T)
		case int32:
			result[i] = any(c.Int32()).(T)
		case int64:
			result[i] = any(c.Int64()).(T)
		case uint:
			result[i] = any(c.Uint()).(T)
		case uint8:
			result[i] = any(c.Uint8()).(T)
		case uint16:
			result[i] = any(c.Uint16()).(T)
		case uint32:
			result[i] = any(c.Uint32()).(T)
		case uint64:
			result[i] = any(c.Uint64()).(T)
		case float32:
			result[i] = any(c.Float32()).(T)
		case float64:
			result[i] = any(c.Float64()).(T)
		case string:
			result[i] = any(c.String()).(T)
		default:
			panic("fuzz.Slice: unsupported element type")
		}
	}
	return result
}
