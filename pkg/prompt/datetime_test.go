package prompt

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDateNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.Date("Select date")

		require.Error(t, err)
	})
}

func TestTimeNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.Time("Select time", 5)

		require.Error(t, err)
	})
}

func TestDateTimeNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.DateTime("Select date and time", 5)

		require.Error(t, err)
	})
}

func TestAdjustViewMonth(t *testing.T) {
	t.Run("returns same month when dates match", func(t *testing.T) {
		viewMonth := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		result := adjustViewMonth(viewMonth, selected)

		require.Equal(t, viewMonth, result)
	})

	t.Run("adjusts to selected month when different", func(t *testing.T) {
		viewMonth := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC)

		result := adjustViewMonth(viewMonth, selected)

		expected := time.Date(2024, 2, 1, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})

	t.Run("adjusts to selected year when different", func(t *testing.T) {
		viewMonth := time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

		result := adjustViewMonth(viewMonth, selected)

		expected := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		require.Equal(t, expected, result)
	})
}

func TestRenderCalendar(t *testing.T) {
	t.Run("renders calendar with header", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		viewMonth := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		lines := p.renderCalendar("Select date", viewMonth, selected)

		require.NotEmpty(t, lines)
		require.Equal(t, "Select date", lines[0])
		require.Contains(t, lines[2], "January")
		require.Contains(t, lines[2], "2024")
		require.Contains(t, lines[3], "Su Mo Tu We Th Fr Sa")
	})

	t.Run("highlights selected date", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		viewMonth := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		lines := p.renderCalendar("Select date", viewMonth, selected)

		found := false
		for _, line := range lines {
			if contains(line, "[15]") {
				found = true
				break
			}
		}
		require.True(t, found, "selected date should be highlighted with brackets")
	})

	t.Run("includes navigation hint", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		viewMonth := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		lines := p.renderCalendar("Select date", viewMonth, selected)

		lastLine := lines[len(lines)-1]
		require.Contains(t, lastLine, "PgUp/PgDn")
		require.Contains(t, lastLine, "Enter")
	})
}

func TestRenderTimePicker(t *testing.T) {
	t.Run("renders time with hour focused", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		lines := p.renderTimePicker("Select time", 14, 30, true)

		require.NotEmpty(t, lines)
		require.Equal(t, "Select time", lines[0])

		found := false
		for _, line := range lines {
			if contains(line, "[14]") {
				found = true
				break
			}
		}
		require.True(t, found, "hour should be highlighted when focused")
	})

	t.Run("renders time with minute focused", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		lines := p.renderTimePicker("Select time", 14, 30, false)

		found := false
		for _, line := range lines {
			if contains(line, "[30]") {
				found = true
				break
			}
		}
		require.True(t, found, "minute should be highlighted when focused")
	})

	t.Run("includes navigation hint", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		lines := p.renderTimePicker("Select time", 14, 30, true)

		lastLine := lines[len(lines)-1]
		require.Contains(t, lastLine, "Tab")
		require.Contains(t, lastLine, "Enter")
	})
}

func TestRenderDateTimePicker(t *testing.T) {
	t.Run("shows date focus indicator when date focused", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		viewMonth := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		lines := p.renderDateTimePicker("Pick", viewMonth, selected, 14, 30, true, true)

		found := false
		for _, line := range lines {
			if contains(line, "[Date]") {
				found = true
				break
			}
		}
		require.True(t, found, "Date should be highlighted when focused")
	})

	t.Run("shows time focus indicator when time focused", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))

		viewMonth := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		selected := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

		lines := p.renderDateTimePicker("Pick", viewMonth, selected, 14, 30, false, true)

		found := false
		for _, line := range lines {
			if contains(line, "[Time]") {
				found = true
				break
			}
		}
		require.True(t, found, "Time should be highlighted when focused")
	})
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}
