package streamschema_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
)

// splitAllMax is the largest input for which every two-chunk split is tried.
const splitAllMax = 512

// mustJSON marshals v. Tests build schemas and instances as typed values;
// raw bytes are used only where the bytes themselves are under test.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

// mustCompile compiles schema with the default limits. A []byte schema is
// used as is; anything else is marshaled first.
func mustCompile(t *testing.T, schema any) *streamschema.Schema {
	t.Helper()
	return mustCompileWith(t, schema, streamschema.DefaultLimits())
}

// mustCompileWith compiles schema under limits.
func mustCompileWith(t *testing.T, schema any, limits streamschema.Limits) *streamschema.Schema {
	t.Helper()
	document, ok := schema.([]byte)
	if !ok {
		document = mustJSON(t, schema)
	}
	compiled, err := streamschema.Compile(document, limits)
	require.NoError(t, err)
	return compiled
}

// validate runs s over input delivered in many chunkings and requires every
// run to reach the same outcome, so no test can pass on one read pattern and
// fail on another. It returns the outcome.
func validate(t *testing.T, s *streamschema.Schema, input []byte) error {
	t.Helper()
	want := s.Validate(context.Background(), bytes.NewReader(input))

	readers := map[string]func() io.Reader{
		"one byte":    func() io.Reader { return iotest.OneByteReader(bytes.NewReader(input)) },
		"half":        func() io.Reader { return iotest.HalfReader(bytes.NewReader(input)) },
		"data+EOF":    func() io.Reader { return iotest.DataErrReader(bytes.NewReader(input)) },
		"stalling":    func() io.Reader { return &stallingReader{data: input, stalls: 3, stalled: 0} },
		"chunks of 7": func() io.Reader { return &chunkReader{data: input, size: 7} },
	}
	for name, reader := range readers {
		requireSameOutcome(t, want, s.Validate(context.Background(), reader()), name)
	}
	if len(input) <= splitAllMax {
		for split := range len(input) + 1 {
			reader := io.MultiReader(bytes.NewReader(input[:split]), bytes.NewReader(input[split:]))
			requireSameOutcome(t, want, s.Validate(context.Background(), reader), "split")
		}
	}
	return want
}

// requireSameOutcome requires got to equal want, field by field for
// validation errors.
func requireSameOutcome(t *testing.T, want, got error, reader string) {
	t.Helper()
	if want == nil {
		require.NoError(t, got, reader)
		return
	}
	var wantFailure, gotFailure *streamschema.ValidationError
	require.ErrorAs(t, want, &wantFailure, reader)
	require.ErrorAs(t, got, &gotFailure, reader)
	require.Equal(t, *wantFailure, *gotFailure, reader)
}

// requireFailure requires err to be a validation error with the given
// reason and locations, and returns its offset.
func requireFailure(t *testing.T, err error, reason streamschema.Reason, instanceLocation, keywordLocation string) int64 {
	t.Helper()
	var failure *streamschema.ValidationError
	require.ErrorAs(t, err, &failure)
	require.Equal(t, reason, failure.Reason, failure.Error())
	require.Equal(t, instanceLocation, failure.InstanceLocation, failure.Error())
	require.Equal(t, keywordLocation, failure.KeywordLocation, failure.Error())
	require.NotEmpty(t, failure.Message)
	return failure.Offset
}

// chunkReader returns data at most size bytes per Read.
type chunkReader struct {
	data []byte
	size int
}

func (r *chunkReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p[:min(len(p), r.size)], r.data)
	r.data = r.data[n:]
	return n, nil
}

// stallingReader returns (0, nil) stalls times before every byte.
type stallingReader struct {
	data    []byte
	stalls  int
	stalled int
}

func (r *stallingReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	if r.stalled < r.stalls {
		r.stalled++
		return 0, nil
	}
	r.stalled = 0
	p[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}

// patternReader yields prefix, then pattern repeated count times, then suffix,
// without materializing the whole stream.
type patternReader struct {
	prefix  []byte
	pattern []byte
	count   int64
	suffix  []byte
	offset  int
}

func (r *patternReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		switch {
		case len(r.prefix) > 0:
			copied := copy(p[n:], r.prefix)
			r.prefix = r.prefix[copied:]
			n += copied
		case r.count > 0:
			copied := copy(p[n:], r.pattern[r.offset:])
			r.offset += copied
			n += copied
			if r.offset == len(r.pattern) {
				r.offset = 0
				r.count--
			}
		case len(r.suffix) > 0:
			copied := copy(p[n:], r.suffix)
			r.suffix = r.suffix[copied:]
			n += copied
		default:
			if n == 0 {
				return 0, io.EOF
			}
			return n, nil
		}
	}
	return n, nil
}

// errBoom is a reader failure that tests look for with errors.Is.
var errBoom = errors.New("boom")
