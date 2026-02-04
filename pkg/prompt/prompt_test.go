package prompt

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// errTestRead is a sentinel error used in tests to simulate read failures.
var errTestRead = errors.New("test read error")

// errorReader is a mock reader that always returns an error.
// Used to test error handling paths in prompt methods.
type errorReader struct {
	err error
}

func (e *errorReader) Read(_ []byte) (int, error) {
	return 0, e.err
}

func TestString(t *testing.T) {
	t.Run("reads trimmed input", func(t *testing.T) {
		in := strings.NewReader("hello world\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Name")

		require.NoError(t, err)
		require.Equal(t, "hello world", result)
		require.Contains(t, out.String(), "Name:")
	})

	t.Run("trims whitespace", func(t *testing.T) {
		in := strings.NewReader("  spaced  \n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Input")

		require.NoError(t, err)
		require.Equal(t, "spaced", result)
	})

	t.Run("handles empty input", func(t *testing.T) {
		in := strings.NewReader("\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Empty")

		require.NoError(t, err)
		require.Equal(t, "", result)
	})

	t.Run("returns default on empty input", func(t *testing.T) {
		in := strings.NewReader("\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Name", "DefaultName")

		require.NoError(t, err)
		require.Equal(t, "DefaultName", result)
		require.Contains(t, out.String(), "[DefaultName]")
	})

	t.Run("overrides default with user input", func(t *testing.T) {
		in := strings.NewReader("CustomName\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Name", "DefaultName")

		require.NoError(t, err)
		require.Equal(t, "CustomName", result)
	})

	t.Run("handles unicode input", func(t *testing.T) {
		in := strings.NewReader("こんにちは世界\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Greeting")

		require.NoError(t, err)
		require.Equal(t, "こんにちは世界", result)
	})

	t.Run("handles emoji input", func(t *testing.T) {
		in := strings.NewReader("Hello 👋 World 🌍\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Message")

		require.NoError(t, err)
		require.Equal(t, "Hello 👋 World 🌍", result)
	})

	t.Run("handles very long input", func(t *testing.T) {
		longString := strings.Repeat("a", 10000)
		in := strings.NewReader(longString + "\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Long")

		require.NoError(t, err)
		require.Equal(t, longString, result)
		require.Len(t, result, 10000)
	})

	t.Run("handles whitespace-only input", func(t *testing.T) {
		in := strings.NewReader("   \t  \n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Whitespace")

		require.NoError(t, err)
		require.Equal(t, "", result)
	})

	t.Run("returns error on EOF without newline", func(t *testing.T) {
		in := strings.NewReader("partial")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("EOF")

		require.Error(t, err)
		require.Equal(t, io.EOF, err)
		require.Equal(t, "partial", result)
	})

	t.Run("returns error on read failure", func(t *testing.T) {
		expectedErr := errors.New("read failed")
		in := &errorReader{err: expectedErr}
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.String("Error")

		require.Error(t, err)
	})

	t.Run("handles special characters", func(t *testing.T) {
		in := strings.NewReader("hello\tworld\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Special")

		require.NoError(t, err)
		require.Equal(t, "hello\tworld", result)
	})

	t.Run("ignores empty default", func(t *testing.T) {
		in := strings.NewReader("value\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.String("Label", "")

		require.NoError(t, err)
		require.Equal(t, "value", result)
		require.NotContains(t, out.String(), "[")
	})
}

func TestInt(t *testing.T) {
	t.Run("parses valid integer", func(t *testing.T) {
		in := strings.NewReader("42\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Int("Age")

		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("parses negative integer", func(t *testing.T) {
		in := strings.NewReader("-10\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Int("Offset")

		require.NoError(t, err)
		require.Equal(t, -10, result)
	})

	t.Run("returns error for non-integer", func(t *testing.T) {
		in := strings.NewReader("abc\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.Int("Number")

		require.Error(t, err)
	})

	t.Run("returns error for float", func(t *testing.T) {
		in := strings.NewReader("3.14\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.Int("Number")

		require.Error(t, err)
	})

	t.Run("returns default on empty input", func(t *testing.T) {
		in := strings.NewReader("\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Int("Age", 30)

		require.NoError(t, err)
		require.Equal(t, 30, result)
		require.Contains(t, out.String(), "[30]")
	})

	t.Run("overrides default with user input", func(t *testing.T) {
		in := strings.NewReader("42\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Int("Age", 30)

		require.NoError(t, err)
		require.Equal(t, 42, result)
	})

	t.Run("accepts human-readable suffix k", func(t *testing.T) {
		in := strings.NewReader("5k\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Int("Limit")

		require.NoError(t, err)
		require.Equal(t, 5000, result)
	})

	t.Run("accepts human-readable suffix m", func(t *testing.T) {
		in := strings.NewReader("2m\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Int("Limit")

		require.NoError(t, err)
		require.Equal(t, 2000000, result)
	})

	t.Run("accepts decimal with suffix", func(t *testing.T) {
		in := strings.NewReader("1.5k\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Int("Limit")

		require.NoError(t, err)
		require.Equal(t, 1500, result)
	})
}

func TestFloat(t *testing.T) {
	t.Run("parses valid float", func(t *testing.T) {
		in := strings.NewReader("3.14\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Float("Value")

		require.NoError(t, err)
		require.InDelta(t, 3.14, result, 0.001)
	})

	t.Run("parses integer as float", func(t *testing.T) {
		in := strings.NewReader("42\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Float("Value")

		require.NoError(t, err)
		require.InDelta(t, 42.0, result, 0.001)
	})

	t.Run("parses negative float", func(t *testing.T) {
		in := strings.NewReader("-2.5\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Float("Value")

		require.NoError(t, err)
		require.InDelta(t, -2.5, result, 0.001)
	})

	t.Run("returns error for non-numeric input", func(t *testing.T) {
		in := strings.NewReader("abc\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.Float("Value")

		require.Error(t, err)
	})

	t.Run("returns default on empty input", func(t *testing.T) {
		in := strings.NewReader("\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Float("Value", 1.5)

		require.NoError(t, err)
		require.InDelta(t, 1.5, result, 0.001)
	})

	t.Run("accepts human-readable suffix", func(t *testing.T) {
		in := strings.NewReader("2.5k\n")
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		result, err := p.Float("Value")

		require.NoError(t, err)
		require.InDelta(t, 2500.0, result, 0.001)
	})
}

func TestSelectNonTTY(t *testing.T) {
	t.Run("returns error when stdin is not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.Select("Pick", map[string]string{"a": "Option A"})

		require.Error(t, err)
	})
}

func TestMultiSelectNonTTY(t *testing.T) {
	t.Run("returns error when stdin is not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}

		p := New(WithReader(in), WithWriter(out))
		_, err := p.MultiSelect("Pick", map[string]string{"a": "Option A"})

		require.Error(t, err)
	})
}

func TestNew(t *testing.T) {
	t.Run("creates prompt with defaults", func(t *testing.T) {
		p := New()

		require.NotNil(t, p)
		require.NotNil(t, p.in)
		require.NotNil(t, p.out)
	})

	t.Run("applies WithReader option", func(t *testing.T) {
		customReader := strings.NewReader("")
		p := New(WithReader(customReader))

		require.Equal(t, customReader, p.in)
	})

	t.Run("applies WithWriter option", func(t *testing.T) {
		customWriter := &bytes.Buffer{}
		p := New(WithWriter(customWriter))

		require.Equal(t, customWriter, p.out)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		customReader := strings.NewReader("")
		customWriter := &bytes.Buffer{}
		p := New(WithReader(customReader), WithWriter(customWriter))

		require.Equal(t, customReader, p.in)
		require.Equal(t, customWriter, p.out)
	})
}
