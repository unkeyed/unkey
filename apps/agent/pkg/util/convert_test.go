package util_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type TestStruct1 struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

type TestStruct2 struct {
	Field3 string `json:"field3"`
	Field4 int    `json:"field4"`
}

type NestedStruct struct {
	Inner TestStruct2 `json:"inner"`
}

func TestStructToMap_NilInput(t *testing.T) {
	result := util.StructToMap(nil)
	require.Empty(t, result)
}

func TestStructToMap_SimpleStruct(t *testing.T) {
	input := TestStruct1{
		Field1: "value1",
		Field2: 42,
	}
	expected := map[string]interface{}{
		"field1": "value1",
		"field2": 42,
	}
	result := util.StructToMap(input)
	require.Equal(t, expected, result)
}

func TestStructToMap_NestedStruct(t *testing.T) {
	input := NestedStruct{
		Inner: TestStruct2{
			Field3: "value3",
			Field4: 99,
		},
	}
	expected := map[string]interface{}{
		"inner": map[string]interface{}{
			"field3": "value3",
			"field4": 99,
		},
	}
	result := util.StructToMap(input)
	require.Equal(t, expected, result)
}
