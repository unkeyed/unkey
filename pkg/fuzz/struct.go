package fuzz

import (
	"reflect"
	"time"
)

// Struct populates a struct of type T with generated values.
//
// Only exported fields are filled. Unexported fields are left at their zero value.
// Nested structs are filled recursively. Slice fields use [Slice] for extraction.
//
// Supported field types are: bool, int, int8, int16, int32, int64, uint, uint8,
// uint16, uint32, uint64, float32, float64, string, time.Time, time.Duration,
// slices of supported types, and nested structs containing supported types.
//
// Skips the test if insufficient bytes remain. Panics if T is not a struct or
// contains unsupported field types.
func Struct[T any](c *Consumer) T {
	var result T
	v := reflect.ValueOf(&result).Elem()
	fillStruct(c, v)
	return result
}

// fillStruct recursively fills a struct's exported fields.
func fillStruct(c *Consumer, v reflect.Value) {
	t := v.Type()

	for i := range v.NumField() {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields
		if !fieldType.IsExported() {
			continue
		}

		fillValue(c, field)
	}
}

// fillValue fills a single reflect.Value with fuzzed data.
//
//nolint:exhaustive // We only support a subset of reflect.Kind; unsupported kinds panic.
func fillValue(c *Consumer, v reflect.Value) {
	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(c.Bool())

	case reflect.Int:
		v.SetInt(int64(c.Int()))
	case reflect.Int8:
		v.SetInt(int64(c.Int8()))
	case reflect.Int16:
		v.SetInt(int64(c.Int16()))
	case reflect.Int32:
		v.SetInt(int64(c.Int32()))
	case reflect.Int64:
		// Handle time.Duration specially
		if v.Type() == reflect.TypeOf(time.Duration(0)) {
			v.SetInt(int64(c.Duration()))
		} else {
			v.SetInt(c.Int64())
		}

	case reflect.Uint:
		v.SetUint(uint64(c.Uint()))
	case reflect.Uint8:
		v.SetUint(uint64(c.Uint8()))
	case reflect.Uint16:
		v.SetUint(uint64(c.Uint16()))
	case reflect.Uint32:
		v.SetUint(uint64(c.Uint32()))
	case reflect.Uint64:
		v.SetUint(c.Uint64())

	case reflect.Float32:
		v.SetFloat(float64(c.Float32()))
	case reflect.Float64:
		v.SetFloat(c.Float64())

	case reflect.String:
		v.SetString(c.String())

	case reflect.Struct:
		// Handle time.Time specially
		if v.Type() == reflect.TypeOf(time.Time{}) {
			v.Set(reflect.ValueOf(c.Time()))
		} else {
			fillStruct(c, v)
		}

	case reflect.Slice:
		fillSlice(c, v)

	default:
		panic("fuzz.Struct: unsupported field type: " + v.Type().String())
	}
}

// fillSlice fills a slice field with fuzzed data.
func fillSlice(c *Consumer, v reflect.Value) {
	length := int(c.Uint8())
	slice := reflect.MakeSlice(v.Type(), length, length)

	for i := range length {
		elem := slice.Index(i)
		fillValue(c, elem)
	}

	v.Set(slice)
}
