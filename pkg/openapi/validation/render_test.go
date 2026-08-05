package validation

import (
	"errors"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/stretchr/testify/require"
)

// Tests for the bounds on anything rendered out of the specification.

// A schema is free to declare a large enum; the response is not free to grow with
// it.
func TestDescribeKindBoundsEnumLists(t *testing.T) {
	t.Parallel()

	want := make([]any, 0, 40)
	for range 40 {
		want = append(want, "value")
	}

	got := describeKind(&kind.Enum{Got: canary, Want: want}, nil)
	require.Equal(t, "must be one of: "+repeatJoin("'value'", maxListedValues)+", …", got.message)
}

func repeatJoin(s string, n int) string {
	out := s
	for range n - 1 {
		out += ", " + s
	}

	return out
}

// A pattern or an enum member long enough to be a payload is dropped rather than
// echoed, in case a future library version starts putting request data in a field
// this package treats as coming from the specification.
func TestDescribeKindDropsOversizedSpecTokens(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", maxSpecTokenLen+1)

	require.Equal(t, "must match the pattern defined in the schema",
		describeKind(&kind.Pattern{Got: canary, Want: long}, nil).message)
	require.Equal(t, "must be one of the values listed in the schema",
		describeKind(&kind.Enum{Got: canary, Want: []any{long}}, nil).message)
	require.Equal(t, "must match the format defined in the schema",
		describeKind(&kind.Format{Got: canary, Want: long, Err: errors.New(canary)}, nil).message)
	require.Equal(t, "is missing properties that the schema requires alongside the ones that were sent",
		describeKind(&kind.DependentRequired{Prop: "card", Missing: []string{long}}, nil).message)
}

// The resolved names come out of a rendered schema, so they are bounded the same
// way an enum list is: a name long enough to be a payload drops the whole hint,
// and a wide object is truncated rather than listed in full.
func TestDescribeKindBoundsAllowedPropertyNames(t *testing.T) {
	t.Parallel()

	oversized := func() []string { return []string{"ok", strings.Repeat("a", maxSpecTokenLen+1)} }
	got := describeKind(&kind.AdditionalProperties{Properties: []string{canary}}, oversized)
	require.Equal(t,
		"Remove any property this operation does not define. The names are withheld because they came from the request.",
		got.fix)

	wide := make([]string, 0, maxListedValues+5)
	for i := range maxListedValues + 5 {
		wide = append(wide, "p"+string(rune('a'+i)))
	}
	got = describeKind(&kind.AdditionalProperties{Properties: []string{canary}}, func() []string { return wide })
	require.Equal(t, "Remove any property that is not one of: "+
		strings.Join(wide[:maxListedValues], ", ")+", ….", got.fix)
}
