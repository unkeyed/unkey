package streamschema_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
)

// requireLimit requires err to be a limit failure at offset.
func requireLimit(t *testing.T, err error, offset int64) {
	t.Helper()
	var failure *streamschema.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, streamschema.ReasonLimit, failure.Reason, failure.Error())
	require.Equal(t, offset, failure.Offset, failure.Error())
}

// TestValidateEnforcesLimitsAtTheirBoundaries guarantees each limit admits
// input exactly at the limit and rejects input one unit past it, before the
// validator allocates or recurses for the excess.
func TestValidateEnforcesLimitsAtTheirBoundaries(t *testing.T) {
	t.Run("DepthMax", func(t *testing.T) {
		s := mustCompile(t, true)
		require.NoError(t, validate(t, s, []byte(strings.Repeat("[", 128)+strings.Repeat("]", 128))))
		requireLimit(t, validate(t, s, []byte(strings.Repeat("[", 129)+strings.Repeat("]", 129))), 128)
		// A million open brackets fail at the limit without deep recursion.
		requireLimit(t, s.Validate(context.Background(), strings.NewReader(strings.Repeat(`{"a":`, 1_000_000))), 128*5)
	})

	t.Run("KeyBytesMax counts decoded bytes", func(t *testing.T) {
		s := mustCompile(t, true)
		require.NoError(t, validate(t, s, mustJSON(t, map[string]int{strings.Repeat("é", 512): 1})))
		requireLimit(t, validate(t, s, mustJSON(t, map[string]int{strings.Repeat("é", 512) + "x": 1})), 1)
		// The bytes matter: 1024 escapes decode to 1024 bytes and fit.
		require.NoError(t, validate(t, s, []byte(`{"`+strings.Repeat(`\u0041`, 1024)+`":1}`)))
	})

	t.Run("NumberBytesMax", func(t *testing.T) {
		s := mustCompile(t, true)
		require.NoError(t, validate(t, s, []byte(strings.Repeat("9", 1024))))
		requireLimit(t, validate(t, s, []byte("["+strings.Repeat("9", 1025)+"]")), 1)
	})

	t.Run("exponent is rejected rather than approximated", func(t *testing.T) {
		s := mustCompile(t, map[string]any{"maximum": 1})
		requireFailure(t, validate(t, s, []byte(`1e1000000000`)), streamschema.ReasonSchema, "", "/maximum")
		require.NoError(t, validate(t, s, []byte(`1e-1000000000`)))
		require.NoError(t, validate(t, s, []byte(`1e-000000000000000000000000001`)))
		requireLimit(t, validate(t, s, []byte(`1e1000000001`)), 0)
		requireLimit(t, validate(t, s, []byte(`[1e-99999999999999999999]`)), 1)
	})

	t.Run("KeysRetainedBytesMax charges every name", func(t *testing.T) {
		limits := streamschema.DefaultLimits()
		limits.KeysRetainedBytesMax = 10 * 65
		s := mustCompileWith(t, true, limits)

		object := func(names int) map[string]int {
			members := make(map[string]int, names)
			for i := range names {
				members[string(rune('a'+i))] = i
			}
			return members
		}
		require.NoError(t, validate(t, s, mustJSON(t, object(10))))
		err := validate(t, s, mustJSON(t, object(11)))
		var failure *streamschema.ValidationError
		require.ErrorAs(t, err, &failure)
		require.Equal(t, streamschema.ReasonLimit, failure.Reason)

		// Names are released when their object closes, so many small objects
		// in sequence stay within the budget.
		many := make([]map[string]int, 1000)
		for i := range many {
			many[i] = object(10)
		}
		require.NoError(t, s.Validate(context.Background(), bytes.NewReader(mustJSON(t, many))))

		// Names of all open objects count together.
		nested := map[string]any{"a": map[string]any{"b": map[string]any{"c": map[string]any{"d": 1, "e": 2, "f": 3, "g": 4, "h": 5, "i": 6, "j": 7, "k": 8}}}}
		err = validate(t, s, mustJSON(t, nested))
		require.ErrorAs(t, err, &failure)
		require.Equal(t, streamschema.ReasonLimit, failure.Reason)
	})

	t.Run("InputBytesMax", func(t *testing.T) {
		limits := streamschema.DefaultLimits()
		limits.InputBytesMax = 16
		s := mustCompileWith(t, true, limits)
		require.NoError(t, validate(t, s, []byte(`"`+strings.Repeat("x", 14)+`"`)))
		requireLimit(t, validate(t, s, []byte(`"`+strings.Repeat("x", 14)+`" `)), 16)

		// An endless body is read no further than one byte past the limit.
		endless := &countingReader{r: &patternReader{prefix: nil, pattern: []byte(" "), count: 1 << 40, suffix: nil, offset: 0}, read: 0, eof: false}
		requireLimit(t, s.Validate(context.Background(), endless), 16)
		require.Equal(t, int64(17), endless.read)
	})
}

// TestValidateConsumesThroughEOF guarantees that a successful validation has
// read the whole body, so a body copied while validating is complete.
func TestValidateConsumesThroughEOF(t *testing.T) {
	s := mustCompile(t, map[string]any{"type": "object"})
	input := []byte("{}\n\n  \t")
	reader := &countingReader{r: bytes.NewReader(input), read: 0, eof: false}

	require.NoError(t, s.Validate(context.Background(), reader))
	require.Equal(t, int64(len(input)), reader.read)
	require.True(t, reader.eof)
}

// TestValidatePreservesReaderAndContextErrors guarantees callers can tell a
// broken connection or a canceled request from invalid input with errors.Is.
func TestValidatePreservesReaderAndContextErrors(t *testing.T) {
	s := mustCompile(t, true)

	t.Run("reader error", func(t *testing.T) {
		reader := io.MultiReader(strings.NewReader(`{"a":"`), iotest.ErrReader(errBoom))
		err := s.Validate(context.Background(), reader)
		require.ErrorIs(t, err, errBoom)
		var failure *streamschema.ValidationError
		require.NotErrorAs(t, err, &failure)
	})

	t.Run("canceled before reading", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		require.ErrorIs(t, s.Validate(ctx, strings.NewReader(`1`)), context.Canceled)
	})

	t.Run("canceled while reading a large buffered body", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		body := &patternReader{prefix: []byte(`[`), pattern: []byte(`1,`), count: 1 << 20, suffix: []byte(`1]`), offset: 0}
		reader := &cancelingReader{r: body, cancel: cancel}
		require.ErrorIs(t, s.Validate(ctx, reader), context.Canceled)
	})

	t.Run("deadline", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		t.Cleanup(cancel)
		require.ErrorIs(t, s.Validate(ctx, strings.NewReader(`1`)), context.DeadlineExceeded)
	})

	t.Run("reader that never makes progress", func(t *testing.T) {
		require.ErrorIs(t, s.Validate(context.Background(), stuckReader{}), io.ErrNoProgress)
	})
}

// TestValidateHeapIsIndependentOfInputSize guarantees the streaming property
// that motivates this package: a huge string or array costs the same heap
// allocations as a small one.
func TestValidateHeapIsIndependentOfInputSize(t *testing.T) {
	stringSchema := mustCompile(t, map[string]any{"type": "object", "properties": map[string]any{"s": map[string]any{"maxLength": 1 << 40}}})
	stringBody := func(count int64) io.Reader {
		// Mixed ASCII, multi-byte UTF-8, and escapes in each repetition.
		return &patternReader{prefix: []byte(`{"s":"`), pattern: []byte(`abcé😀\n\ud83d\ude00\"`), count: count, suffix: []byte(`"}`), offset: 0}
	}
	arraySchema := mustCompile(t, map[string]any{"items": map[string]any{"type": "integer", "maximum": 1000}})
	arrayBody := func(count int64) io.Reader {
		return &patternReader{prefix: []byte(`[`), pattern: []byte(`123,`), count: count, suffix: []byte(`1]`), offset: 0}
	}

	for name, run := range map[string]struct {
		schema *streamschema.Schema
		body   func(int64) io.Reader
	}{
		"string": {schema: stringSchema, body: stringBody},
		"array":  {schema: arraySchema, body: arrayBody},
	} {
		t.Run(name, func(t *testing.T) {
			allocs := func(count int64) float64 {
				return testing.AllocsPerRun(3, func() {
					require.NoError(t, run.schema.Validate(context.Background(), run.body(count)))
				})
			}
			small, large := allocs(1<<10), allocs(1<<21)
			require.Equal(t, small, large, "allocations grew with input size")
		})
	}

	// Length is counted in code points across the whole stream: 8 per
	// repetition, since each escape and the surrogate pair count as one.
	limited := mustCompile(t, map[string]any{"properties": map[string]any{"s": map[string]any{"maxLength": 8 * (1 << 20)}}})
	require.NoError(t, limited.Validate(context.Background(), stringBody(1<<20)))
	limited = mustCompile(t, map[string]any{"properties": map[string]any{"s": map[string]any{"maxLength": 8*(1<<20) - 1}}})
	requireFailure(t, limited.Validate(context.Background(), stringBody(1<<20)), streamschema.ReasonSchema, "/s", "/properties/s/maxLength")
}

// TestSchemaIsSafeForConcurrentUse guarantees a compiled schema can be shared
// by concurrent requests.
func TestSchemaIsSafeForConcurrentUse(t *testing.T) {
	s := mustCompile(t, map[string]any{
		"properties": map[string]any{"n": map[string]any{"type": "integer", "minimum": 0}},
		"required":   []string{"n"},
	})

	var wg sync.WaitGroup
	errs := make([]error, 64)
	for i := range errs {
		wg.Go(func() {
			errs[i] = s.Validate(context.Background(), strings.NewReader(fmt.Sprintf(`{"n":%d}`, i-32)))
		})
	}
	wg.Wait()

	for i, err := range errs {
		if i < 32 {
			requireFailure(t, err, streamschema.ReasonSchema, "/n", "/properties/n/minimum")
		} else {
			require.NoError(t, err)
		}
	}
}

// countingReader counts bytes read and records whether EOF was returned.
type countingReader struct {
	r    io.Reader
	read int64
	eof  bool
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.read += int64(n)
	if errors.Is(err, io.EOF) {
		r.eof = true
	}
	return n, err
}

// cancelingReader cancels its context after the first read.
type cancelingReader struct {
	r      io.Reader
	cancel context.CancelFunc
}

func (r *cancelingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.cancel()
	return n, err
}

// stuckReader always returns (0, nil).
type stuckReader struct{}

func (stuckReader) Read([]byte) (int, error) {
	return 0, nil
}
