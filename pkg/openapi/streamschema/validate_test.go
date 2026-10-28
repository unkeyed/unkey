package streamschema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
)

// schemaCase is one instance checked against one schema. A failure is
// identified by its locations; nil wantKeyword means the instance is valid.
type schemaCase struct {
	name            string
	schema          any
	instance        any
	instanceLocator string
	wantKeyword     *string
}

func keyword(location string) *string {
	return &location
}

// TestValidateAppliesSupportedKeywords guarantees the JSON Schema 2020-12
// meaning of every supported keyword, including that type-specific keywords
// ignore instances of other types and that failures name the exact instance
// and keyword.
func TestValidateAppliesSupportedKeywords(t *testing.T) {
	type object = map[string]any
	type array = []any
	number := func(literal string) json.Number { return json.Number(literal) }

	cases := []schemaCase{
		{name: "true schema accepts anything", schema: true, instance: object{"a": array{1, "x", nil}}},
		{name: "false schema rejects anything", schema: false, instance: nil, wantKeyword: keyword("")},
		{name: "type string accepts string", schema: object{"type": "string"}, instance: "x"},
		{name: "type string rejects number", schema: object{"type": "string"}, instance: 1, wantKeyword: keyword("/type")},
		{name: "type list admits null", schema: object{"type": array{"string", "null"}}, instance: nil},
		{name: "type list rejects boolean", schema: object{"type": array{"string", "null"}}, instance: true, wantKeyword: keyword("/type")},
		{name: "integer accepts 1.0", schema: object{"type": "integer"}, instance: number("1.0")},
		{name: "integer accepts 1e0", schema: object{"type": "integer"}, instance: number("1e0")},
		{name: "integer accepts 10.5e1", schema: object{"type": "integer"}, instance: number("10.5e1")},
		{name: "integer accepts -0", schema: object{"type": "integer"}, instance: number("-0")},
		{name: "integer rejects 1.5", schema: object{"type": "integer"}, instance: number("1.5"), wantKeyword: keyword("/type")},
		{name: "integer rejects 1e-1", schema: object{"type": "integer"}, instance: number("1e-1"), wantKeyword: keyword("/type")},
		{name: "number admits integers", schema: object{"type": "number"}, instance: 3},
		{name: "maxLength ignores numbers", schema: object{"maxLength": 2}, instance: 12345},
		{name: "minimum ignores strings", schema: object{"minimum": 5}, instance: "x"},
		{name: "properties ignore arrays", schema: object{"properties": object{"a": false}}, instance: array{1}},
		{name: "items ignore objects", schema: object{"items": false}, instance: object{"a": 1}},
		{name: "required ignores strings", schema: object{"required": array{"a"}}, instance: "a"},

		{name: "maxLength counts code points", schema: object{"maxLength": 2}, instance: "é😀"},
		{name: "maxLength rejects longer", schema: object{"maxLength": 2}, instance: "abc", wantKeyword: keyword("/maxLength")},
		{name: "minLength rejects shorter", schema: object{"minLength": 2}, instance: "😀", wantKeyword: keyword("/minLength")},

		{name: "properties apply by name", schema: object{"properties": object{"a": object{"type": "string"}}}, instance: object{"a": 1}, instanceLocator: "/a", wantKeyword: keyword("/properties/a/type")},
		{name: "additionalProperties false rejects extra", schema: object{"properties": object{"a": true}, "additionalProperties": false}, instance: object{"a": 1, "b": 2}, instanceLocator: "/b", wantKeyword: keyword("/additionalProperties")},
		{name: "additionalProperties schema applies to extra only", schema: object{"properties": object{"a": true}, "additionalProperties": object{"type": "string"}}, instance: object{"a": 1, "b": "x"}},
		{name: "required rejects missing", schema: object{"required": array{"a", "b"}}, instance: object{"a": 1}, wantKeyword: keyword("/required")},
		{name: "required accepts present null", schema: object{"required": array{"a"}}, instance: object{"a": nil}},
		{name: "pointer escapes member names", schema: object{"additionalProperties": false}, instance: object{"a/b~c": 1}, instanceLocator: "/a~1b~0c", wantKeyword: keyword("/additionalProperties")},

		{name: "items apply to every element", schema: object{"items": object{"properties": object{"n": object{"maxLength": 1}}}}, instance: array{object{"n": "a"}, object{"n": "ab"}}, instanceLocator: "/1/n", wantKeyword: keyword("/items/properties/n/maxLength")},
		{name: "items false accepts empty array", schema: object{"items": false}, instance: array{}},
		{name: "items false rejects first element", schema: object{"items": false}, instance: array{1}, instanceLocator: "/0", wantKeyword: keyword("/items")},
		{name: "maxItems rejects extra", schema: object{"maxItems": 1}, instance: array{1, 2}, wantKeyword: keyword("/maxItems")},
		{name: "minItems rejects fewer", schema: object{"minItems": 2}, instance: array{1}, wantKeyword: keyword("/minItems")},

		{name: "maximum int64 accepts itself", schema: []byte(`{"maximum":9223372036854775807}`), instance: number("9223372036854775807")},
		{name: "maximum int64 accepts scientific equal", schema: []byte(`{"maximum":9223372036854775807}`), instance: number("9.223372036854775807e18")},
		{name: "maximum int64 rejects next integer", schema: []byte(`{"maximum":9223372036854775807}`), instance: number("9223372036854775808"), wantKeyword: keyword("/maximum")},
		{name: "maximum int64 rejects tiny excess", schema: []byte(`{"maximum":9223372036854775807}`), instance: number("9223372036854775807.0000000000000000001"), wantKeyword: keyword("/maximum")},
		{name: "minimum 0.25 rejects just below", schema: object{"minimum": number("0.25")}, instance: number("0.2499999999999999999999"), wantKeyword: keyword("/minimum")},
		{name: "minimum 0.25 accepts 25e-2", schema: object{"minimum": number("0.25")}, instance: number("25e-2")},
		{name: "minimum negative rejects lower", schema: object{"minimum": number("-1.5")}, instance: number("-1.50001"), wantKeyword: keyword("/minimum")},
		{name: "minimum negative accepts higher", schema: object{"minimum": number("-1.5")}, instance: number("-1.49")},
		{name: "exclusiveMinimum rejects equal zero", schema: object{"exclusiveMinimum": 0}, instance: number("-0"), wantKeyword: keyword("/exclusiveMinimum")},
		{name: "exclusiveMinimum accepts tiny positive", schema: object{"exclusiveMinimum": 0}, instance: number("1e-400")},
		{name: "exclusiveMaximum rejects equal", schema: object{"exclusiveMaximum": number("1e2")}, instance: number("100.0"), wantKeyword: keyword("/exclusiveMaximum")},
		{name: "maximum beyond float64 compares exactly", schema: object{"maximum": number("1e400")}, instance: number("9.99e399")},
		{name: "maximum beyond float64 rejects larger", schema: object{"maximum": number("1e400")}, instance: number("1.0000000001e400"), wantKeyword: keyword("/maximum")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validate(t, mustCompile(t, tc.schema), mustJSON(t, tc.instance))
			if tc.wantKeyword == nil {
				require.NoError(t, err)
				return
			}
			requireFailure(t, err, streamschema.ReasonSchema, tc.instanceLocator, *tc.wantKeyword)
		})
	}
}

// TestValidateAcceptsIntegerValuedCountKeywords guarantees that count
// keywords written as integer-valued decimals, as in the JSON Schema Test
// Suite, behave like their integer forms.
func TestValidateAcceptsIntegerValuedCountKeywords(t *testing.T) {
	// The schema bytes matter: json.Marshal would print 2.0 as 2.
	s := mustCompile(t, []byte(`{"maxLength":2.0,"minItems":2.0,"maxItems":3e0}`))

	require.NoError(t, validate(t, s, mustJSON(t, "ab")))
	requireFailure(t, validate(t, s, mustJSON(t, "abc")), streamschema.ReasonSchema, "", "/maxLength")
	requireFailure(t, validate(t, s, mustJSON(t, []int{1})), streamschema.ReasonSchema, "", "/minItems")
	requireFailure(t, validate(t, s, mustJSON(t, []int{1, 2, 3, 4})), streamschema.ReasonSchema, "", "/maxItems")
}

// TestValidateReportsOffsets guarantees offsets point at the start of the
// failing token, so callers can locate failures in huge bodies.
func TestValidateReportsOffsets(t *testing.T) {
	s := mustCompile(t, map[string]any{
		"properties": map[string]any{"a": map[string]any{"maxItems": 1}},
		"required":   []string{"z"},
	})

	// The bytes matter: offsets are asserted.
	offset := requireFailure(t, validate(t, s, []byte(`{"a": [1, 2]}`)), streamschema.ReasonSchema, "/a", "/properties/a/maxItems")
	require.Equal(t, int64(10), offset)

	offset = requireFailure(t, validate(t, s, []byte(`  {"a": []}`)), streamschema.ReasonSchema, "", "/required")
	require.Equal(t, int64(2), offset)
}

// TestValidateRequiresStrictSyntax guarantees the validator rejects every
// input that a lenient downstream decoder could read differently than the
// validator did: malformed JSON, invalid UTF-8, unpaired surrogates, byte
// order marks, duplicate names, and trailing data.
func TestValidateRequiresStrictSyntax(t *testing.T) {
	valid := []string{
		`{"a":[1,2,{"b":null}],"c":true,"d":false}`,
		" \t\n\r1 \t\n\r",
		`"\u00e9\/\b\f\n\r\t\"\\"`,
		`"\ud83d\ude00"`,
		`"\uD83D\uDE00"`,
		"\"é😀\x7f\"",
		`[{"a":1},{"a":1}]`,
		`{"a":{"a":1}}`,
		`{"":1}`,
		`-0.0e-0`,
		`1E+2`,
	}
	for _, input := range valid {
		t.Run("valid "+input, func(t *testing.T) {
			require.NoError(t, validate(t, mustCompile(t, true), []byte(input)))
		})
	}

	invalid := []struct {
		input  string
		offset int64
	}{
		{input: ``, offset: 0},
		{input: `   `, offset: 3},
		{input: `{`, offset: 1},
		{input: `[1,]`, offset: 3},
		{input: `{"a":1,}`, offset: 7},
		{input: `[1 2]`, offset: 3},
		{input: `{"a" 1}`, offset: 5},
		{input: `{'a':1}`, offset: 1},
		{input: `01`, offset: 0},
		{input: `1.`, offset: 0},
		{input: `.5`, offset: 0},
		{input: `-`, offset: 0},
		{input: `+1`, offset: 0},
		{input: `1e`, offset: 0},
		{input: `NaN`, offset: 0},
		{input: `Infinity`, offset: 0},
		{input: `tru`, offset: 0},
		{input: `nul`, offset: 0},
		{input: `truex`, offset: 4},
		{input: `"abc`, offset: 0},
		{input: "\"a\x01\"", offset: 2},
		{input: `"\u12"`, offset: 1},
		{input: `"\x"`, offset: 1},
		{input: `"\ud800"`, offset: 1},
		{input: `"\udc00"`, offset: 1},
		{input: `"\ud800\u0041"`, offset: 1},
		{input: `"\ud800\udbff"`, offset: 1},
		{input: "\"\xff\"", offset: 1},
		{input: "\"\xc0\xaf\"", offset: 1},
		{input: "\"\xed\xa0\x80\"", offset: 1},
		{input: "\"\xe2\x82\"", offset: 1},
		{input: "\xef\xbb\xbf{}", offset: 0},
		{input: `1 2`, offset: 2},
		{input: `{}x`, offset: 2},
		{input: `{"a":1,"a":2}`, offset: 7},
		{input: `{"a":1,"\u0061":2}`, offset: 7},
	}
	for _, tc := range invalid {
		t.Run("invalid "+tc.input, func(t *testing.T) {
			err := validate(t, mustCompile(t, true), []byte(tc.input))
			var failure *streamschema.ValidationError
			require.ErrorAs(t, err, &failure)
			require.Equal(t, streamschema.ReasonSyntax, failure.Reason, failure.Error())
			require.Equal(t, tc.offset, failure.Offset, failure.Error())
		})
	}
}

// TestValidateReportsSyntaxLocation guarantees syntax failures name the
// member being read, which helps clients find malformed parts of large bodies.
func TestValidateReportsSyntaxLocation(t *testing.T) {
	// The bytes matter: the input is malformed on purpose.
	err := validate(t, mustCompile(t, true), []byte(`{"a":[1,{"b":tru}]}`))
	requireFailure(t, err, streamschema.ReasonSyntax, "/a/1/b", "")
}

func TestValidateDoesNotEchoStringValues(t *testing.T) {
	secret := strings.Repeat("s3cr3t", 4)
	s := mustCompile(t, map[string]any{"maxLength": 3, "type": "string"})

	err := validate(t, s, mustJSON(t, secret))
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
}
