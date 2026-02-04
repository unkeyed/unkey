package prompt

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHumanInt(t *testing.T) {
	t.Run("parses plain integers", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"0", 0},
			{"1", 1},
			{"-1", -1},
			{"42", 42},
			{"-42", -42},
			{"1000", 1000},
			{"007", 7},
		}

		for _, tt := range tests {
			result, err := parseHumanInt(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.expected, result, "input: %s", tt.input)
		}
	})

	t.Run("parses thousands with k suffix", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"1k", 1000},
			{"1K", 1000},
			{"10k", 10000},
			{"100k", 100000},
			{"-1k", -1000},
			{"0k", 0},
		}

		for _, tt := range tests {
			result, err := parseHumanInt(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.expected, result, "input: %s", tt.input)
		}
	})

	t.Run("parses millions with m suffix", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"1m", 1000000},
			{"1M", 1000000},
			{"10m", 10000000},
			{"-1m", -1000000},
		}

		for _, tt := range tests {
			result, err := parseHumanInt(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.expected, result, "input: %s", tt.input)
		}
	})

	t.Run("parses billions with b suffix", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"1b", 1000000000},
			{"1B", 1000000000},
			{"2b", 2000000000},
			{"-1b", -1000000000},
		}

		for _, tt := range tests {
			result, err := parseHumanInt(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.expected, result, "input: %s", tt.input)
		}
	})

	t.Run("parses trillions with t suffix", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"1t", 1000000000000},
			{"1T", 1000000000000},
			{"-1t", -1000000000000},
		}

		for _, tt := range tests {
			result, err := parseHumanInt(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.expected, result, "input: %s", tt.input)
		}
	})

	t.Run("parses decimals with suffix resulting in whole number", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"1.5k", 1500},
			{"2.5k", 2500},
			{"0.5k", 500},
			{"1.5m", 1500000},
			{"-1.5k", -1500},
		}

		for _, tt := range tests {
			result, err := parseHumanInt(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.expected, result, "input: %s", tt.input)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"  42  ", 42},
			{" 1k ", 1000},
			{"\t100\t", 100},
			{"  5k  ", 5000},
		}

		for _, tt := range tests {
			result, err := parseHumanInt(tt.input)
			require.NoError(t, err, "input: %q", tt.input)
			require.Equal(t, tt.expected, result, "input: %q", tt.input)
		}
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		_, err := parseHumanInt("")
		require.Error(t, err)

		_, err = parseHumanInt("   ")
		require.Error(t, err)
	})

	t.Run("returns error for invalid input", func(t *testing.T) {
		invalidInputs := []string{
			"abc",
			"12abc",
			"1.2.3",
			"--1",
			"1kk",
			"k",
			"1 000",
		}

		for _, input := range invalidInputs {
			_, err := parseHumanInt(input)
			require.Error(t, err, "expected error for input: %q", input)
		}
	})

	t.Run("returns error for decimal without suffix", func(t *testing.T) {
		_, err := parseHumanInt("1.5")
		require.Error(t, err)

		_, err = parseHumanInt("3.14")
		require.Error(t, err)
	})

	t.Run("returns error for non-whole result", func(t *testing.T) {
		// 1.1111k = 1111.1 (not a whole number)
		_, err := parseHumanInt("1.1111k")
		require.Error(t, err)

		// 1.23456k = 1234.56 (not a whole number)
		_, err = parseHumanInt("1.23456k")
		require.Error(t, err)
	})

	t.Run("returns error for overflow", func(t *testing.T) {
		// Way beyond int64 max
		_, err := parseHumanInt("9999999999999999999999")
		require.Error(t, err)

		// 9999999t = 9.999999e18, exceeds int64 max (~9.2e18)
		_, err = parseHumanInt("9999999t")
		require.Error(t, err)

		_, err = parseHumanInt("-9999999999999999999999")
		require.Error(t, err)
	})

	t.Run("returns error for suffix without number", func(t *testing.T) {
		_, err := parseHumanInt("k")
		require.Error(t, err)

		_, err = parseHumanInt("  m")
		require.Error(t, err)
	})

	t.Run("handles edge cases at int bounds", func(t *testing.T) {
		result, err := parseHumanInt("9223372036854775807")
		require.NoError(t, err)
		require.Equal(t, 9223372036854775807, result)

		result, err = parseHumanInt("-9223372036854775808")
		require.NoError(t, err)
		require.Equal(t, -9223372036854775808, result)
	})
}

func TestParseHumanFloat(t *testing.T) {
	t.Run("parses plain floats", func(t *testing.T) {
		tests := []struct {
			input    string
			expected float64
		}{
			{"0", 0},
			{"1", 1},
			{"-1", -1},
			{"3.14", 3.14},
			{"-3.14", -3.14},
			{"0.5", 0.5},
			{"100.001", 100.001},
		}

		for _, tt := range tests {
			result, err := parseHumanFloat(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.InDelta(t, tt.expected, result, 0.0001, "input: %s", tt.input)
		}
	})

	t.Run("parses floats with k suffix", func(t *testing.T) {
		tests := []struct {
			input    string
			expected float64
		}{
			{"1k", 1000},
			{"1.5k", 1500},
			{"2.5k", 2500},
			{"0.001k", 1},
			{"-1.5k", -1500},
		}

		for _, tt := range tests {
			result, err := parseHumanFloat(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.InDelta(t, tt.expected, result, 0.0001, "input: %s", tt.input)
		}
	})

	t.Run("parses floats with m suffix", func(t *testing.T) {
		tests := []struct {
			input    string
			expected float64
		}{
			{"1m", 1000000},
			{"1.5m", 1500000},
			{"0.5m", 500000},
		}

		for _, tt := range tests {
			result, err := parseHumanFloat(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.InDelta(t, tt.expected, result, 0.0001, "input: %s", tt.input)
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result, err := parseHumanFloat("  3.14  ")
		require.NoError(t, err)
		require.InDelta(t, 3.14, result, 0.0001)

		result, err = parseHumanFloat(" 1.5k ")
		require.NoError(t, err)
		require.InDelta(t, 1500, result, 0.0001)
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		_, err := parseHumanFloat("")
		require.Error(t, err)

		_, err = parseHumanFloat("   ")
		require.Error(t, err)
	})

	t.Run("returns error for invalid input", func(t *testing.T) {
		invalidInputs := []string{
			"abc",
			"1.2.3",
			"--1",
			"1kk",
			"k",
		}

		for _, input := range invalidInputs {
			_, err := parseHumanFloat(input)
			require.Error(t, err, "expected error for input: %q", input)
		}
	})
}
