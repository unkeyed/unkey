package redaction

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests for which member a path matches: depth, array steps, escaped names, and
// the limits of position tracking.

func TestPaths(t *testing.T) {
	t.Parallel()

	require.Equal(t, []string{"a.b", "key", "x[].y"}, New([]string{"x[].y", "key", "a.b"}).Paths())
	require.Nil(t, (*Redactor)(nil).Paths())
}

// The reason paths exist rather than names: two members called value, one
// annotated and one not, have to come out differently.
func TestRedact_PathAnchoring(t *testing.T) {
	t.Parallel()

	body := []byte(`{"value":"SENSITIVE","xyz":{"value":"HARMLESS"}}`)

	root := New([]string{"value"})
	require.Equal(t,
		`{"value":"[REDACTED]","xyz":{"value":"HARMLESS"}}`,
		string(root.Redact(body)),
	)

	nested := New([]string{"xyz.value"})
	require.Equal(t,
		`{"value":"SENSITIVE","xyz":{"value":"[REDACTED]"}}`,
		string(nested.Redact(body)),
	)

	both := New([]string{"value", "xyz.value"})
	require.Equal(t,
		`{"value":"[REDACTED]","xyz":{"value":"[REDACTED]"}}`,
		string(both.Redact(body)),
	)
}

func TestRedact_PathsMustMatchDepthAndShape(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		in   string
		want string
	}{
		{
			name: "root path does not match a nested member",
			path: "key",
			in:   `{"a":{"key":"visible"}}`,
			want: `{"a":{"key":"visible"}}`,
		},
		{
			name: "nested path does not match at the root",
			path: "a.key",
			in:   `{"key":"visible"}`,
			want: `{"key":"visible"}`,
		},
		{
			name: "array step is required to enter elements",
			path: "variables.value",
			in:   `{"variables":[{"value":"visible"}]}`,
			want: `{"variables":[{"value":"visible"}]}`,
		},
		{
			name: "array step matches every element",
			path: "variables[].value",
			in:   `{"variables":[{"value":"a"},{"value":"b"},{"other":"visible"}]}`,
			want: `{"variables":[{"value":"[REDACTED]"},{"value":"[REDACTED]"},{"other":"visible"}]}`,
		},
		{
			name: "nested arrays need a step each",
			path: "a[][].secret",
			in:   `{"a":[[{"secret":"x"}]]}`,
			want: `{"a":[[{"secret":"[REDACTED]"}]]}`,
		},
		{
			name: "body that is itself an array",
			path: "[].key",
			in:   `[{"key":"a"},{"key":"b"}]`,
			want: `[{"key":"[REDACTED]"},{"key":"[REDACTED]"}]`,
		},
		{
			name: "same name at two depths, only the annotated one goes",
			path: "data.token",
			in:   `{"token":"visible","data":{"token":"secret"}}`,
			want: `{"token":"visible","data":{"token":"[REDACTED]"}}`,
		},
		{
			name: "sibling objects do not inherit a match",
			path: "a.key",
			in:   `{"a":{"key":"secret"},"b":{"key":"visible"}}`,
			want: `{"a":{"key":"[REDACTED]"},"b":{"key":"visible"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, string(New([]string{tt.path}).Redact([]byte(tt.in))))
		})
	}
}

// Found by an adversarial review. A member name may be escaped, and every JSON
// parser in the request path decodes it: the validator accepts the request, the
// handler reads the field, and only the redactor was comparing raw bytes. A caller
// could therefore rename a field past the path tree and have its own credential
// written to the log on a fully successful request.
func TestRedact_EscapedMemberNames(t *testing.T) {
	t.Parallel()

	r := New([]string{"key", "data.key", "variables[].value"})

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "escaped leaf name",
			in:   `{"key":"` + secret + `"}`,
			want: `{"key":"[REDACTED]"}`,
		},
		{
			name: "escaped intermediate name",
			in:   `{"data":{"key":"` + secret + `"}}`,
			want: `{"data":{"key":"[REDACTED]"}}`,
		},
		{
			name: "escaped array-bearing name",
			in:   `{"variables":[{"value":"` + secret + `"}]}`,
			want: `{"variables":[{"value":"[REDACTED]"}]}`,
		},
		{
			name: "decoy alongside the escaped form",
			in:   `{"key":"a","key":"` + secret + `"}`,
			want: `{"key":"[REDACTED]","key":"[REDACTED]"}`,
		},
		{
			name: "escaped name that is not sensitive stays",
			in:   `{"keyId":"key_123"}`,
			want: `{"keyId":"key_123"}`,
		},
		{
			name: "an escape that cannot be decoded is treated as sensitive",
			in:   `{"ke\qy":"` + secret + `"}`,
			want: `{"ke\qy":"[REDACTED]"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(r.Redact([]byte(tt.in)))
			require.Equal(t, tt.want, got)
			require.NotContains(t, got, secret)
		})
	}
}

// Found by an adversarial review. Past maxDepth the scanner cannot track position,
// but that is different from position being wrong: treating it as corruption
// dropped the whole rest of the body to name matching, which silently over-redacted
// unrelated fields in a valid document.
func TestRedact_PastDepthLimitDoesNotDesync(t *testing.T) {
	t.Parallel()

	r := New([]string{"key"})

	deep := strings.Repeat(`{"a":`, 70) + `[[0]]` + strings.Repeat(`}`, 70)
	in := `{"d":` + deep + `,"nested":{"key":"KEEP-ME"}}`

	got := string(r.Redact([]byte(in)))
	require.Contains(t, got, `"key":"KEEP-ME"`, "a nested key that matches no path must survive")
	require.Equal(t, in, got)
}
