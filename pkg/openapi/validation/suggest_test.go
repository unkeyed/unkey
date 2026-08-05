package validation

import (
	"net/http"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/stretchr/testify/require"
)

func TestNearestDeclared(t *testing.T) {
	t.Parallel()

	declared := []string{"name", "externalId", "permissions", "ratelimits"}

	tests := []struct {
		name    string
		sent    []string
		want    string
		suggest bool
	}{
		{name: "trailing colon", sent: []string{"name:"}, want: "name", suggest: true},
		{name: "dropped letter", sent: []string{"nam"}, want: "name", suggest: true},
		{name: "wrong case", sent: []string{"Name"}, want: "name", suggest: true},
		{name: "transposed letters", sent: []string{"exteranlId"}, want: "externalId", suggest: true},
		{name: "plural slip", sent: []string{"permission"}, want: "permissions", suggest: true},
		{name: "one of several sent names matches", sent: []string{"zzz", "ratelimit"}, want: "ratelimits", suggest: true},
		{name: "unrelated name suggests nothing", sent: []string{"extra"}, suggest: false},
		{name: "a credential suggests nothing", sent: []string{"sk_live_2cGKbMxRjIzhCxo"}, suggest: false},
		{name: "empty", sent: nil, suggest: false},
		{name: "no declared names", sent: []string{"name:"}, suggest: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			candidates := declared
			if tt.name == "no declared names" {
				candidates = nil
			}

			got, ok := nearestDeclared(tt.sent, candidates)
			require.Equal(t, tt.suggest, ok)
			if tt.suggest {
				require.Equal(t, tt.want, got)
			}
		})
	}
}

// A short name reaches too many other short names at distance 2, so the bound
// tightens rather than offering a confident wrong answer.
func TestNearestDeclaredIsConservative(t *testing.T) {
	t.Parallel()

	_, ok := nearestDeclared([]string{"xyz"}, []string{"abc"})
	require.False(t, ok, "two unrelated three-letter names must not match")

	// Equidistant candidates are ambiguous, so neither is offered.
	_, ok = nearestDeclared([]string{"nome"}, []string{"name", "node"})
	require.False(t, ok, "a tie must not produce a suggestion")

	// An exact match is not a typo; it is a different failure entirely.
	_, ok = nearestDeclared([]string{"name"}, []string{"name"})
	require.False(t, ok)
}

func TestUndeclaredDescription(t *testing.T) {
	t.Parallel()

	allowed := func() []string { return []string{"name"} }

	one := describeKind(&kind.AdditionalProperties{Properties: []string{"name:"}}, allowed)
	require.Equal(t, "contains 1 property that is not defined in the schema. Did you mean 'name'?", one.message)
	require.Contains(t, one.fix, "Rename it to 'name'")

	two := describeKind(&kind.AdditionalProperties{Properties: []string{"aaa", "bbb"}}, allowed)
	require.Equal(t, "contains 2 properties that are not defined in the schema", two.message)
	require.Contains(t, two.fix, "not one of: name")

	none := describeKind(&kind.AdditionalProperties{Properties: nil}, allowed)
	require.Equal(t, "contains properties that are not defined in the schema", none.message)
}

// The suggestion is a name the schema declares, so it is safe to print. What the
// caller sent is read to compute it and must never reach the output, including
// when the typo is a near miss for a real property.
func TestSuggestionNeverEchoesWhatWasSent(t *testing.T) {
	t.Parallel()

	spec := []byte(`
openapi: 3.0.3
info: {title: probe, version: 1.0.0}
paths:
  /probe:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              properties:
                name: {type: string}
      responses: {"200": {description: ok}}
`)

	v, err := NewFromBytes(spec)
	require.NoError(t, err)

	for _, sent := range []string{"name:", "nam", "Name", canary, "name" + canary} {
		r, err := http.NewRequest(http.MethodPost, "https://x/probe",
			strings.NewReader(`{"`+sent+`":"x"}`))
		require.NoError(t, err)
		r.Header.Set("Content-Type", "application/json")

		result := v.Validate(r)
		require.NotNil(t, result)

		// An echo would be quoted, the way the library used to print it. A bare
		// substring is not enough to assert on, because the suggestion 'name'
		// legitimately contains "nam".
		require.NotContains(t, result.Summary(), "'"+sent+"'", "the property name that was sent came back")
		require.NotContains(t, result.Summary(), canary)
	}
}
