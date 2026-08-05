package validation

import (
	"errors"
	"golang.org/x/text/language"
	"math/big"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/message"
)

// Every error kind the JSON Schema library can produce, each one built with the
// canary in every field that describes what the caller sent. The message is
// asserted exactly, so a wording change is visible, and the canary check below
// makes a leak a test failure rather than a review question.
//
// The list is exhaustive against jsonschema v6.0.2's kind package. When an
// upgrade adds a kind, describeKind routes it to fallbackMessage; the last tests
// in this file are what make that safe.
func TestDescribeKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    jsonschema.ErrorKind
		message string
		fix     bool
		prune   bool
		skip    bool
	}{
		// Grouping nodes: their children carry the real reason, and 'allOf' and
		// '$ref' merge subschemas rather than choosing between them, so the child is
		// not an alternative that the caller was free to ignore.
		{name: "schema", kind: &kind.Schema{Location: canary}, skip: true},
		{name: "group", kind: &kind.Group{}, skip: true},
		{name: "allOf", kind: &kind.AllOf{}, skip: true},
		{name: "reference", kind: &kind.Reference{Keyword: "$ref", URL: canary}, skip: true},
		{
			name: "refCycle",
			kind: &kind.RefCycle{URL: canary, KeywordLocation1: canary, KeywordLocation2: canary},
			skip: true,
		},

		// Branching combinators: reported, and their branch failures dropped.
		{
			name:    "anyOf",
			kind:    &kind.AnyOf{},
			message: "must match at least one of the schemas defined for it",
			prune:   true,
		},
		{
			name:    "oneOf none matched",
			kind:    &kind.OneOf{Subschemas: nil},
			message: "must match exactly one of the schemas defined for it, but it matched none",
			prune:   true,
		},
		{
			name:    "oneOf several matched",
			kind:    &kind.OneOf{Subschemas: []int{0, 2}},
			message: "must match exactly one of the schemas defined for it, but it matched more than one",
			prune:   true,
		},
		{
			name:    "not",
			kind:    &kind.Not{},
			message: "must not match the schema defined under 'not'",
			prune:   true,
		},

		{name: "falseSchema", kind: &kind.FalseSchema{}, message: "is not accepted here by the specification"},

		{
			name:    "type",
			kind:    &kind.Type{Got: "number", Want: []string{"string"}},
			message: "must be a string, but a number was sent",
		},
		{
			name:    "type with several options",
			kind:    &kind.Type{Got: "object", Want: []string{"string", "integer", "null"}},
			message: "must be a string, an integer, or null, but an object was sent",
		},
		{
			name:    "type with an unrecognised got",
			kind:    &kind.Type{Got: canary, Want: []string{"string"}},
			message: "must be a string",
		},
		{
			name:    "type with an unrecognised want",
			kind:    &kind.Type{Got: "string", Want: []string{canary}},
			message: "is not of a type this field accepts",
		},

		{
			name:    "enum",
			kind:    &kind.Enum{Got: canary, Want: []any{"alpha", "beta"}},
			message: "must be one of: 'alpha', 'beta'",
		},
		{
			name:    "enum of mixed scalars",
			kind:    &kind.Enum{Got: canary, Want: []any{float64(1), true, nil}},
			message: "must be one of: 1, true, null",
		},
		{
			name:    "enum of composites",
			kind:    &kind.Enum{Got: canary, Want: []any{map[string]any{"a": 1}}},
			message: "must be one of the values listed in the schema",
		},
		{
			name:    "enum with no values",
			kind:    &kind.Enum{Got: canary, Want: nil},
			message: "must be one of the values listed in the schema",
		},

		{name: "const", kind: &kind.Const{Got: canary, Want: "fixed"}, message: "must be 'fixed'"},
		{
			name:    "const of a composite",
			kind:    &kind.Const{Got: canary, Want: []any{1}},
			message: "must equal the constant value defined in the schema",
		},

		{
			name:    "format",
			kind:    &kind.Format{Got: canary, Want: "date-time", Err: errors.New(canary)},
			message: "must be a valid date-time value",
		},
		{
			name:    "format with no name",
			kind:    &kind.Format{Got: canary, Want: "", Err: errors.New(canary)},
			message: "must match the format defined in the schema",
		},

		{name: "required", kind: &kind.Required{Missing: []string{"a", "b"}}, message: "is required"},

		{
			name:    "dependency",
			kind:    &kind.Dependency{Prop: "card", Missing: []string{"cvc", "expiry"}},
			message: "must also contain 'cvc' and 'expiry' when 'card' is present",
		},
		{
			name:    "dependentRequired",
			kind:    &kind.DependentRequired{Prop: "card", Missing: []string{"cvc"}},
			message: "must also contain 'cvc' when 'card' is present",
		},
		{
			name:    "dependentRequired with nothing declared",
			kind:    &kind.DependentRequired{Prop: "card", Missing: nil},
			message: "is missing properties that the schema requires alongside the ones that were sent",
		},

		{
			name:    "additionalProperties",
			kind:    &kind.AdditionalProperties{Properties: []string{canary, canary + "2"}},
			message: "contains 2 properties that are not defined in the schema",
			fix:     true,
		},
		{
			name:    "propertyNames",
			kind:    &kind.PropertyNames{Property: canary},
			message: "contains a property whose name the schema does not allow",
			fix:     true,
		},

		{name: "minProperties", kind: &kind.MinProperties{Got: 99, Want: 2}, message: "must have at least 2 properties"},
		{name: "maxProperties", kind: &kind.MaxProperties{Got: 99, Want: 1}, message: "must have at most 1 property"},
		{name: "minItems", kind: &kind.MinItems{Got: 99, Want: 3}, message: "must contain at least 3 items"},
		{name: "maxItems", kind: &kind.MaxItems{Got: 99, Want: 1}, message: "must contain at most 1 item"},
		{
			name:    "additionalItems",
			kind:    &kind.AdditionalItems{Count: 7},
			message: "must not contain more items than the schema defines",
		},
		{
			name:    "uniqueItems",
			kind:    &kind.UniqueItems{Duplicates: [2]int{2, 5}},
			message: "must not contain duplicate items, but the items at index 2 and 5 are equal",
		},
		{name: "contains", kind: &kind.Contains{}, message: "must contain at least one item matching the schema"},
		{
			name:    "minContains",
			kind:    &kind.MinContains{Got: []int{1, 2}, Want: 4},
			message: "must contain at least 4 items matching the schema",
		},
		{
			name:    "maxContains",
			kind:    &kind.MaxContains{Got: []int{1, 2, 3}, Want: 2},
			message: "must contain at most 2 items matching the schema",
		},

		{name: "minLength", kind: &kind.MinLength{Got: 0, Want: 1}, message: "must be at least 1 character long"},
		{
			name:    "maxLength",
			kind:    &kind.MaxLength{Got: 16385, Want: 16384},
			message: "must be at most 16384 characters long",
		},
		{
			name:    "pattern",
			kind:    &kind.Pattern{Got: canary, Want: "^[a-z]+$"},
			message: "must match the pattern '^[a-z]+$'",
		},
		{
			name:    "contentEncoding",
			kind:    &kind.ContentEncoding{Want: "base64", Err: errors.New(canary)},
			message: "must be encoded as base64",
		},
		{
			name:    "contentMediaType",
			kind:    &kind.ContentMediaType{Got: []byte(canary), Want: "application/json", Err: errors.New(canary)},
			message: "must decode to media type application/json",
		},
		{
			name:    "contentSchema",
			kind:    &kind.ContentSchema{},
			message: "must decode to a value matching the schema",
		},

		{
			name:    "minimum",
			kind:    &kind.Minimum{Got: big.NewRat(3, 1), Want: big.NewRat(100, 1)},
			message: "must be at least 100",
		},
		{
			name:    "maximum",
			kind:    &kind.Maximum{Got: big.NewRat(424242, 1), Want: big.NewRat(10, 1)},
			message: "must be at most 10",
		},
		{
			name:    "maximum of a fraction",
			kind:    &kind.Maximum{Got: big.NewRat(9, 1), Want: big.NewRat(1, 2)},
			message: "must be at most 0.5",
		},
		{
			name:    "exclusiveMinimum",
			kind:    &kind.ExclusiveMinimum{Got: big.NewRat(1, 1), Want: big.NewRat(0, 1)},
			message: "must be greater than 0",
		},
		{
			name:    "exclusiveMaximum",
			kind:    &kind.ExclusiveMaximum{Got: big.NewRat(9, 1), Want: big.NewRat(5, 1)},
			message: "must be less than 5",
		},
		{
			name:    "multipleOf",
			kind:    &kind.MultipleOf{Got: big.NewRat(7, 1), Want: big.NewRat(5, 1)},
			message: "must be a multiple of 5",
		},

		{
			name:    "invalidJsonValue",
			kind:    &kind.InvalidJsonValue{Value: canary},
			message: "is not a value that can be validated as JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := describeKind(tt.kind, nil)

			if tt.skip {
				require.True(t, got.skip, "expected a grouping node to be skipped")

				return
			}

			require.False(t, got.skip)
			require.Equal(t, tt.message, got.message)
			require.Equal(t, tt.fix, got.fix != "", "unexpected fix hint: %q", got.fix)
			require.Equal(t, tt.prune, got.prune, "unexpected pruning decision")

			require.NotContains(t, got.message, canary)
			require.NotContains(t, got.fix, canary)

			// The library's own rendering of the same failure is what used to be
			// returned. Where it differs, the difference is the request data that is
			// now dropped.
			require.NotEqual(t, tt.kind.LocalizedString(message.NewPrinter(language.English)), got.message)
		})
	}
}

// A kind this package has never seen must degrade to a fixed sentence rather than
// to whatever the library decided to put in its message.
func TestDescribeKindFallsBackForUnknownKinds(t *testing.T) {
	t.Parallel()

	got := describeKind(unknownKind{}, nil)
	require.False(t, got.skip)
	require.False(t, got.prune)
	require.Equal(t, fallbackMessage, got.message)
	require.NotContains(t, got.message, canary)
}

type unknownKind struct{}

func (unknownKind) KeywordPath() []string { return []string{"somethingNew"} }

func (unknownKind) LocalizedString(*message.Printer) string {
	return "the value '" + canary + "' upset the validator in a brand new way"
}

// The two keywords that reject a property the caller named are the only ones that
// can say what would have been accepted, and the names come from the schema. When
// the list is unavailable the hint has to still be useful without it.
func TestDescribeKindListsAllowedPropertyNames(t *testing.T) {
	t.Parallel()

	names := func(list ...string) propertyLookup {
		return func() []string { return list }
	}

	tests := []struct {
		name    string
		kind    jsonschema.ErrorKind
		allowed propertyLookup
		fix     string
	}{
		{
			name:    "additionalProperties with a resolved list",
			kind:    &kind.AdditionalProperties{Properties: []string{canary}},
			allowed: names("name", "apiId", "byteLength"),
			fix:     "Remove any property that is not one of: apiId, byteLength, name.",
		},
		{
			name:    "additionalProperties with nothing resolved",
			kind:    &kind.AdditionalProperties{Properties: []string{canary}},
			allowed: names(),
			fix:     "Remove any property this operation does not define. The names are withheld because they came from the request.",
		},
		{
			name:    "additionalProperties with no resolver at all",
			kind:    &kind.AdditionalProperties{Properties: []string{canary}},
			allowed: nil,
			fix:     "Remove any property this operation does not define. The names are withheld because they came from the request.",
		},
		{
			name:    "propertyNames with a resolved list",
			kind:    &kind.PropertyNames{Property: canary},
			allowed: names("beta", "alpha"),
			fix:     "Rename the property to one of: alpha, beta.",
		},
		{
			name:    "propertyNames with nothing resolved",
			kind:    &kind.PropertyNames{Property: canary},
			allowed: nil,
			fix:     "Rename the property so that its name satisfies the schema. The name is withheld because it came from the request.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := describeKind(tt.kind, tt.allowed)
			require.Equal(t, tt.fix, got.fix)
			require.NotContains(t, got.fix, canary)
		})
	}
}
