package redaction

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests for the scanner against input it cannot trust: truncated, desynchronized,
// hostile, or simply not JSON.

// Truncated bodies are the case a naive redactor fails open on: the value is
// there but its terminator is not, so a scanner that requires well-formed JSON
// skips it and logs the secret.
func TestRedact_TruncatedInputFailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
	}{
		{name: "cut mid string value", in: `{"key":"` + secret},
		{name: "cut before closing quote", in: `{"variables":[{"value":"` + secret},
		{name: "cut mid object value", in: `{"value":{"nested":"` + secret + `"`},
		{name: "cut mid array value", in: `{"value":["` + secret},
		{name: "value missing entirely", in: `{"key":}`},
		{name: "unterminated name", in: `{"key`},
	}

	r := testRedactor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(r.Redact([]byte(tt.in)))
			require.NotContains(t, got, secret, "truncated body leaked the secret: %s", got)
		})
	}
}

// A 1 MiB body is the largest zen captures, so it is the worst case that
// reaches the redactor in production.
func TestRedact_HostileOneMebibyteBody(t *testing.T) {
	t.Parallel()

	r := testRedactor()

	t.Run("secret buried after padding", func(t *testing.T) {
		t.Parallel()
		padding := strings.Repeat("a", 1<<20)
		body := `{"pad":"` + padding + `","key":"` + secret + `"}`
		got := string(r.Redact([]byte(body)))
		require.NotContains(t, got, secret)
		require.Contains(t, got, `"key":"[REDACTED]"`)
	})

	t.Run("oversized secret value", func(t *testing.T) {
		t.Parallel()
		body := `{"key":"` + secret + strings.Repeat("b", 1<<20) + `"}`
		got := string(r.Redact([]byte(body)))
		require.Equal(t, `{"key":"[REDACTED]"}`, got)
	})

	t.Run("deeply nested redacted object", func(t *testing.T) {
		t.Parallel()
		depth := 10000
		body := `{"value":` + strings.Repeat(`{"a":`, depth) + `"` + secret + `"` + strings.Repeat(`}`, depth) + `}`
		got := string(r.Redact([]byte(body)))
		require.Equal(t, `{"value":"[REDACTED]"}`, got)
	})

	t.Run("truncated 1 MiB body", func(t *testing.T) {
		t.Parallel()
		body := `{"key":"` + secret + strings.Repeat("c", 1<<20)
		got := string(r.Redact([]byte(body)))
		require.NotContains(t, got, secret)
	})
}

// Known limitation, pinned so it stays a decision rather than a surprise: with
// no colon anywhere, there is no member to redact, and both strings are values
// as far as any parser is concerned.
func TestRedact_PairlessStringsAreValues(t *testing.T) {
	t.Parallel()

	in := `{"key" "` + secret + `"}`
	require.Equal(t, in, string(testRedactor().Redact([]byte(in))))
}

// Found by FuzzRedact. A colon after a value means the value was really the
// next name in a chain, so the redaction has to follow it to the end.
func TestRedact_ColonChain(t *testing.T) {
	t.Parallel()

	r := testRedactor()

	require.Equal(t, `{"key":"[REDACTED]"}`, string(r.Redact([]byte(`{"key":"key":"`+secret+`"}`))))
	require.Equal(t, `{"key":"[REDACTED]"}`, string(r.Redact([]byte(`{"key":"a":"b":"`+secret+`"}`))))
	require.Equal(t, `{"key":"[REDACTED]","name":"visible"}`, string(r.Redact([]byte(`{"key":"a":{"b":"`+secret+`"},"name":"visible"}`))))
}

// Found by FuzzRedact. An odd number of quotes before a real member shifts every
// following string token by one, so a scanner that trusts its own alignment
// reads `":"` as the token and walks straight past the secret.
func TestRedact_ResynchronizesAfterStrayQuotes(t *testing.T) {
	t.Parallel()

	tests := []string{
		`""key":"` + secret + `"`,
		`"{"key":"` + secret + `"}`,
		`x"{"variables":[{"value":"` + secret + `"}]}`,
		`""""key":"` + secret + `"`,
	}

	r := testRedactor()
	for _, in := range tests {
		got := string(r.Redact([]byte(in)))
		require.NotContains(t, got, secret, "stray quotes hid the secret in %q: %s", in, got)
	}
}

func FuzzRedact(f *testing.F) {
	seeds := []string{
		`{"key":"` + secret + `"}`,
		`{"value":{"a":[1,2,"` + secret + `"]}}`,
		`{"key":"a\"b"}`,
		`{"keyId":"key_123"}`,
		`{"key":"` + secret,
		`{"key":}`,
		`{}`,
		``,
		`not json`,
		`{"key":"\\"}`,
		`""key":"` + secret + `"`,
		`"{"key":"` + secret + `"}`,
		`"key":"key":"` + secret + `"`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	r := testRedactor()

	f.Fuzz(func(t *testing.T, in []byte) {
		original := string(in)
		out := r.Redact(in)

		require.Equal(t, original, string(in), "input was mutated")
		require.Equal(t, string(out), string(r.Redact(out)), "not idempotent for input %q", in)
		if json.Valid(in) {
			require.True(t, json.Valid(out), "valid input %q produced invalid output %q", in, out)
		}

		// The guarantee the redactor actually makes is about well-formed bodies
		// and truncated prefixes of them, so build one around the fuzz input
		// and require the redacted member's value to be gone. Arbitrary garbage
		// is not covered: bytes like `"key":"a"key":"s"` leave the trailing
		// value governed by nothing, which is documented on Redact.
		//
		// The canary is distinct from secret because the fuzz input lands in a
		// member nobody marked sensitive, where a secret is expected to survive.
		const canary = "CANARY_9f31c7a2"
		body, err := json.Marshal(map[string]any{"noise": string(in), "key": canary})
		if err != nil {
			return
		}
		require.NotContains(t, string(r.Redact(body)), canary, "leaked from well-formed body %q", body)

		// Truncation at any point must not expose it either. Sampling a few
		// cut points keeps this linear per input.
		for _, cut := range []int{len(body) / 2, len(body) * 3 / 4, len(body) - 1} {
			if cut <= 0 {
				continue
			}
			require.NotContains(t, string(r.Redact(body[:cut])), canary, "leaked from body truncated at %d: %q", cut, body[:cut])
		}
	})
}
