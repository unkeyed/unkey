package prompt

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)

	t.Run("parses today", func(t *testing.T) {
		result, err := ParseDate("today")
		require.NoError(t, err)
		require.Equal(t, today, result)
	})

	t.Run("parses TODAY (case insensitive)", func(t *testing.T) {
		result, err := ParseDate("TODAY")
		require.NoError(t, err)
		require.Equal(t, today, result)
	})

	t.Run("parses tomorrow", func(t *testing.T) {
		result, err := ParseDate("tomorrow")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(0, 0, 1), result)
	})

	t.Run("parses yesterday", func(t *testing.T) {
		result, err := ParseDate("yesterday")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(0, 0, -1), result)
	})

	t.Run("parses +1d relative", func(t *testing.T) {
		result, err := ParseDate("+1d")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(0, 0, 1), result)
	})

	t.Run("parses -1d relative", func(t *testing.T) {
		result, err := ParseDate("-1d")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(0, 0, -1), result)
	})

	t.Run("parses +2w relative", func(t *testing.T) {
		result, err := ParseDate("+2w")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(0, 0, 14), result)
	})

	t.Run("parses -1w relative", func(t *testing.T) {
		result, err := ParseDate("-1w")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(0, 0, -7), result)
	})

	t.Run("parses +3m relative", func(t *testing.T) {
		result, err := ParseDate("+3m")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(0, 3, 0), result)
	})

	t.Run("parses -1y relative", func(t *testing.T) {
		result, err := ParseDate("-1y")
		require.NoError(t, err)
		require.Equal(t, today.AddDate(-1, 0, 0), result)
	})

	t.Run("parses ISO format YYYY-MM-DD", func(t *testing.T) {
		result, err := ParseDate("2024-01-15")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses US format MM/DD/YYYY", func(t *testing.T) {
		result, err := ParseDate("01/15/2024")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses US format with single digits M/D/YYYY", func(t *testing.T) {
		result, err := ParseDate("1/5/2024")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 5, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses European format DD.MM.YYYY", func(t *testing.T) {
		result, err := ParseDate("15.01.2024")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses two-digit year", func(t *testing.T) {
		result, err := ParseDate("01/15/24")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		result, err := ParseDate("  today  ")
		require.NoError(t, err)
		require.Equal(t, today, result)
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		_, err := ParseDate("")
		require.Error(t, err)

		_, err = ParseDate("   ")
		require.Error(t, err)
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		invalidInputs := []string{
			"abc",
			"2024",
			"2024-13-01",
			"++1d",
			"+1x",
			"+d",
		}
		for _, input := range invalidInputs {
			_, err := ParseDate(input)
			require.Error(t, err, "expected error for input: %q", input)
		}
	})
}

func TestParseTime(t *testing.T) {
	t.Run("parses now", func(t *testing.T) {
		now := time.Now()
		hour, minute, err := ParseTime("now")
		require.NoError(t, err)
		require.Equal(t, now.Hour(), hour)
		require.Equal(t, now.Minute(), minute)
	})

	t.Run("parses 24-hour format HH:MM", func(t *testing.T) {
		tests := []struct {
			input  string
			hour   int
			minute int
		}{
			{"00:00", 0, 0},
			{"14:30", 14, 30},
			{"23:59", 23, 59},
			{"09:05", 9, 5},
		}
		for _, tt := range tests {
			hour, minute, err := ParseTime(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.hour, hour, "input: %s", tt.input)
			require.Equal(t, tt.minute, minute, "input: %s", tt.input)
		}
	})

	t.Run("parses 12-hour format with pm", func(t *testing.T) {
		tests := []struct {
			input  string
			hour   int
			minute int
		}{
			{"2:30pm", 14, 30},
			{"2:30PM", 14, 30},
			{"2:30 pm", 14, 30},
			{"12:00pm", 12, 0},
			{"12:30pm", 12, 30},
		}
		for _, tt := range tests {
			hour, minute, err := ParseTime(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.hour, hour, "input: %s", tt.input)
			require.Equal(t, tt.minute, minute, "input: %s", tt.input)
		}
	})

	t.Run("parses 12-hour format with am", func(t *testing.T) {
		tests := []struct {
			input  string
			hour   int
			minute int
		}{
			{"9:30am", 9, 30},
			{"9:30AM", 9, 30},
			{"12:00am", 0, 0},
			{"12:30am", 0, 30},
		}
		for _, tt := range tests {
			hour, minute, err := ParseTime(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.hour, hour, "input: %s", tt.input)
			require.Equal(t, tt.minute, minute, "input: %s", tt.input)
		}
	})

	t.Run("parses military time HHMM", func(t *testing.T) {
		tests := []struct {
			input  string
			hour   int
			minute int
		}{
			{"0000", 0, 0},
			{"1430", 14, 30},
			{"2359", 23, 59},
		}
		for _, tt := range tests {
			hour, minute, err := ParseTime(tt.input)
			require.NoError(t, err, "input: %s", tt.input)
			require.Equal(t, tt.hour, hour, "input: %s", tt.input)
			require.Equal(t, tt.minute, minute, "input: %s", tt.input)
		}
	})

	t.Run("parses hour only", func(t *testing.T) {
		hour, minute, err := ParseTime("14")
		require.NoError(t, err)
		require.Equal(t, 14, hour)
		require.Equal(t, 0, minute)

		hour, minute, err = ParseTime("2pm")
		require.NoError(t, err)
		require.Equal(t, 14, hour)
		require.Equal(t, 0, minute)
	})

	t.Run("trims whitespace", func(t *testing.T) {
		hour, minute, err := ParseTime("  14:30  ")
		require.NoError(t, err)
		require.Equal(t, 14, hour)
		require.Equal(t, 30, minute)
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		_, _, err := ParseTime("")
		require.Error(t, err)
	})

	t.Run("returns error for invalid format", func(t *testing.T) {
		invalidInputs := []string{
			"abc",
			"25:00",
			"12:60",
			"-1:00",
		}
		for _, input := range invalidInputs {
			_, _, err := ParseTime(input)
			require.Error(t, err, "expected error for input: %q", input)
		}
	})
}

func TestParseDateTime(t *testing.T) {
	t.Run("parses ISO format with space", func(t *testing.T) {
		result, err := ParseDateTime("2024-01-15 14:30")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses ISO format with T separator", func(t *testing.T) {
		result, err := ParseDateTime("2024-01-15T14:30")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses today with time", func(t *testing.T) {
		now := time.Now()
		result, err := ParseDateTime("today 14:30")
		require.NoError(t, err)
		expected := time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses tomorrow with 12-hour time", func(t *testing.T) {
		now := time.Now()
		tomorrow := now.AddDate(0, 0, 1)
		result, err := ParseDateTime("tomorrow 2pm")
		require.NoError(t, err)
		expected := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 14, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses relative date with time", func(t *testing.T) {
		now := time.Now()
		future := now.AddDate(0, 0, 1)
		result, err := ParseDateTime("+1d 9:00am")
		require.NoError(t, err)
		expected := time.Date(future.Year(), future.Month(), future.Day(), 9, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses date only", func(t *testing.T) {
		result, err := ParseDateTime("2024-01-15")
		require.NoError(t, err)
		expected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("parses time only", func(t *testing.T) {
		now := time.Now()
		result, err := ParseDateTime("14:30")
		require.NoError(t, err)
		expected := time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		_, err := ParseDateTime("")
		require.Error(t, err)
	})

	t.Run("returns error for invalid input", func(t *testing.T) {
		_, err := ParseDateTime("invalid datetime")
		require.Error(t, err)
	})
}
