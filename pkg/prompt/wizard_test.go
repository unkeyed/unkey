package prompt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWizard(t *testing.T) {
	t.Run("creates wizard with correct total", func(t *testing.T) {
		p := New()
		wiz := p.Wizard(5)

		require.Equal(t, 5, wiz.Total())
		require.Equal(t, 0, wiz.Current())
	})

	t.Run("Skip advances current step", func(t *testing.T) {
		p := New()
		wiz := p.Wizard(3)

		require.Equal(t, 0, wiz.Current())
		wiz.Skip()
		require.Equal(t, 1, wiz.Current())
		wiz.Skip()
		require.Equal(t, 2, wiz.Current())
	})
}

func TestWizardPrefix(t *testing.T) {
	t.Run("shows correct progress for first step", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))
		wiz := p.Wizard(3)

		prefix := wiz.prefix()

		require.Contains(t, prefix, "[")
		require.Contains(t, prefix, "]")
		require.Contains(t, prefix, "●")
		require.Contains(t, prefix, "○")
	})

	t.Run("advances prefix after successful prompt", func(t *testing.T) {
		in := strings.NewReader("test\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(3)

		require.Equal(t, 0, wiz.Current())

		_, err := wiz.String("Name")
		require.NoError(t, err)
		require.Equal(t, 1, wiz.Current())
	})
}

func TestWizardString(t *testing.T) {
	t.Run("includes progress prefix in output", func(t *testing.T) {
		in := strings.NewReader("hello\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		result, err := wiz.String("Name")

		require.NoError(t, err)
		require.Equal(t, "hello", result)
		require.Contains(t, out.String(), "[")
		require.Contains(t, out.String(), "]")
		require.Contains(t, out.String(), "Name")
	})

	t.Run("advances on success", func(t *testing.T) {
		in := strings.NewReader("test\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(3)

		_, err := wiz.String("Step 1")
		require.NoError(t, err)
		require.Equal(t, 1, wiz.Current())
	})

	t.Run("does not advance on error", func(t *testing.T) {
		in := &errorReader{err: errTestRead}
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(3)

		_, err := wiz.String("Step 1")
		require.Error(t, err)
		require.Equal(t, 0, wiz.Current())
	})

	t.Run("uses default value", func(t *testing.T) {
		in := strings.NewReader("\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		result, err := wiz.String("Name", "DefaultName")

		require.NoError(t, err)
		require.Equal(t, "DefaultName", result)
	})
}

func TestWizardInt(t *testing.T) {
	t.Run("parses integer with progress prefix", func(t *testing.T) {
		in := strings.NewReader("42\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		result, err := wiz.Int("Age")

		require.NoError(t, err)
		require.Equal(t, 42, result)
		require.Contains(t, out.String(), "Age")
	})

	t.Run("accepts human-readable suffix", func(t *testing.T) {
		in := strings.NewReader("1k\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		result, err := wiz.Int("Limit")

		require.NoError(t, err)
		require.Equal(t, 1000, result)
	})

	t.Run("advances on success", func(t *testing.T) {
		in := strings.NewReader("100\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(3)

		_, err := wiz.Int("Step 1")
		require.NoError(t, err)
		require.Equal(t, 1, wiz.Current())
	})
}

func TestWizardFloat(t *testing.T) {
	t.Run("parses float with progress prefix", func(t *testing.T) {
		in := strings.NewReader("3.14\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		result, err := wiz.Float("Value")

		require.NoError(t, err)
		require.InDelta(t, 3.14, result, 0.001)
	})

	t.Run("accepts human-readable suffix", func(t *testing.T) {
		in := strings.NewReader("1.5k\n")
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		result, err := wiz.Float("Value")

		require.NoError(t, err)
		require.InDelta(t, 1500.0, result, 0.001)
	})
}

func TestWizardSelectNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		_, err := wiz.Select("Pick", map[string]string{"a": "A"})

		require.Error(t, err)
		require.Equal(t, 0, wiz.Current())
	})
}

func TestWizardMultiSelectNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		_, err := wiz.MultiSelect("Pick", map[string]string{"a": "A"})

		require.Error(t, err)
		require.Equal(t, 0, wiz.Current())
	})
}

func TestWizardDateNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		_, err := wiz.Date("Pick date")

		require.Error(t, err)
		require.Equal(t, 0, wiz.Current())
	})
}

func TestWizardTimeNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		_, err := wiz.Time("Pick time", 5)

		require.Error(t, err)
		require.Equal(t, 0, wiz.Current())
	})
}

func TestWizardDateTimeNonTTY(t *testing.T) {
	t.Run("returns error when not a TTY", func(t *testing.T) {
		in := bytes.NewReader([]byte{})
		out := &bytes.Buffer{}
		p := New(WithReader(in), WithWriter(out))
		wiz := p.Wizard(2)

		_, err := wiz.DateTime("Pick datetime", 15)

		require.Error(t, err)
		require.Equal(t, 0, wiz.Current())
	})
}

func TestWizardDone(t *testing.T) {
	t.Run("prints message in green", func(t *testing.T) {
		out := &bytes.Buffer{}
		p := New(WithWriter(out))
		wiz := p.Wizard(1)

		wiz.Done("All done!")

		output := out.String()
		require.Contains(t, output, "All done!")
		require.Contains(t, output, colorGreen)
		require.Contains(t, output, colorReset)
	})
}

func TestWizardMultipleSteps(t *testing.T) {
	t.Run("completes full workflow", func(t *testing.T) {
		out := &bytes.Buffer{}

		in1 := strings.NewReader("step1\n")
		p := New(WithReader(in1), WithWriter(out))
		wiz := p.Wizard(3)

		r1, err := wiz.String("First")
		require.NoError(t, err)
		require.Equal(t, "step1", r1)
		require.Equal(t, 1, wiz.Current())

		in2 := strings.NewReader("step2\n")
		wiz.prompt.in = in2

		r2, err := wiz.String("Second")
		require.NoError(t, err)
		require.Equal(t, "step2", r2)
		require.Equal(t, 2, wiz.Current())

		in3 := strings.NewReader("step3\n")
		wiz.prompt.in = in3

		r3, err := wiz.String("Third")
		require.NoError(t, err)
		require.Equal(t, "step3", r3)
		require.Equal(t, 3, wiz.Current())
	})

	t.Run("handles skip in workflow", func(t *testing.T) {
		out := &bytes.Buffer{}

		in1 := strings.NewReader("first\n")
		p := New(WithReader(in1), WithWriter(out))
		wiz := p.Wizard(3)

		r1, err := wiz.String("First")
		require.NoError(t, err)
		require.Equal(t, "first", r1)

		wiz.Skip()
		require.Equal(t, 2, wiz.Current())

		in3 := strings.NewReader("third\n")
		wiz.prompt.in = in3

		r3, err := wiz.String("Third")
		require.NoError(t, err)
		require.Equal(t, "third", r3)
		require.Equal(t, 3, wiz.Current())
	})
}
