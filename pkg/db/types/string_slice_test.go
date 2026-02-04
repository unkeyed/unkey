package dbtype

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringSlice_ScanAndValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected StringSlice
	}{
		{"nil", nil, StringSlice{}},
		{"empty bytes", []byte{}, StringSlice{}},
		{"empty array", []byte("[]"), StringSlice{}},
		{"single element", []byte(`["foo"]`), StringSlice{"foo"}},
		{"multiple elements", []byte(`["foo","bar","baz"]`), StringSlice{"foo", "bar", "baz"}},
		{"string input", `["a","b"]`, StringSlice{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s StringSlice
			err := s.Scan(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, s)

			// Round-trip through Value
			val, err := s.Value()
			require.NoError(t, err)

			var s2 StringSlice
			err = s2.Scan(val)
			require.NoError(t, err)
			require.Equal(t, s, s2)
		})
	}
}

func TestStringSlice_Value_NilReturnsEmptyArray(t *testing.T) {
	var s StringSlice
	val, err := s.Value()
	require.NoError(t, err)
	require.Equal(t, "[]", val)
}

func TestStringSlice_JSON(t *testing.T) {
	tests := []struct {
		name     string
		input    StringSlice
		expected string
	}{
		{"nil slice", nil, "[]"},
		{"empty slice", StringSlice{}, "[]"},
		{"with elements", StringSlice{"a", "b"}, `["a","b"]`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(data))

			var s StringSlice
			err = json.Unmarshal(data, &s)
			require.NoError(t, err)
			if tt.input == nil {
				require.Equal(t, StringSlice{}, s)
			} else {
				require.Equal(t, tt.input, s)
			}
		})
	}
}

func TestStringSlice_UnmarshalNull(t *testing.T) {
	var s StringSlice
	err := json.Unmarshal([]byte("null"), &s)
	require.NoError(t, err)
	require.Equal(t, StringSlice{}, s)
}
