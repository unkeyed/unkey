package streamschema_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
)

// requireCompileError requires Compile to reject document with kind at
// location, and returns the error by value.
func requireCompileError(t *testing.T, document []byte, limits streamschema.Limits, kind streamschema.CompileErrorKind, location string) streamschema.CompileError {
	t.Helper()
	compiled, err := streamschema.Compile(document, limits)
	require.Nil(t, compiled)
	var compileErr *streamschema.CompileError
	require.ErrorAs(t, err, &compileErr)
	require.Equal(t, kind, compileErr.Kind, compileErr.Error())
	require.Equal(t, location, compileErr.KeywordLocation, compileErr.Error())
	require.NotEmpty(t, compileErr.Message)
	return *compileErr
}

// TestCompileReportsUnsupportedKeywords guarantees that no assertion outside
// the supported subset is silently dropped: every such keyword, unknown
// keyword, or unsupported form fails compilation as unsupported, at its exact
// location, wherever it is nested.
func TestCompileReportsUnsupportedKeywords(t *testing.T) {
	keywords := map[string]any{
		"$ref":                  "#/$defs/a",
		"$defs":                 map[string]any{"a": true},
		"definitions":           map[string]any{"a": true},
		"$id":                   "https://example.com/s",
		"$anchor":               "a",
		"$dynamicRef":           "#a",
		"$dynamicAnchor":        "a",
		"$vocabulary":           map[string]any{},
		"enum":                  []any{1},
		"const":                 1,
		"pattern":               "^a$",
		"multipleOf":            2,
		"allOf":                 []any{true},
		"anyOf":                 []any{true},
		"oneOf":                 []any{true},
		"not":                   false,
		"if":                    true,
		"then":                  true,
		"else":                  true,
		"dependentRequired":     map[string]any{},
		"dependentSchemas":      map[string]any{},
		"uniqueItems":           true,
		"prefixItems":           []any{true},
		"contains":              true,
		"minContains":           1,
		"maxContains":           1,
		"unevaluatedItems":      false,
		"unevaluatedProperties": false,
		"patternProperties":     map[string]any{},
		"propertyNames":         true,
		"minProperties":         1,
		"maxProperties":         1,
		"contentEncoding":       "base64",
		"contentMediaType":      "application/json",
		"contentSchema":         true,
		"nullable":              true,
		"discriminator":         map[string]any{"propertyName": "kind"},
		"xml":                   map[string]any{},
		"x-unkey-redact":        true,
		"maxLenght":             3,
	}
	for name, value := range keywords {
		t.Run(name, func(t *testing.T) {
			document := mustJSON(t, map[string]any{
				"properties": map[string]any{"a/b": map[string]any{name: value}},
			})
			location := "/properties/a~1b/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(name)
			requireCompileError(t, document, streamschema.DefaultLimits(), streamschema.CompileUnsupported, location)
		})
	}

	forms := []struct {
		name     string
		schema   map[string]any
		location string
	}{
		{name: "array form of items", schema: map[string]any{"items": []any{true}}, location: "/items"},
		{name: "OpenAPI 3.0 exclusiveMinimum", schema: map[string]any{"minimum": 1, "exclusiveMinimum": true}, location: "/exclusiveMinimum"},
		{name: "OpenAPI 3.0 exclusiveMaximum", schema: map[string]any{"exclusiveMaximum": false}, location: "/exclusiveMaximum"},
		{name: "older dialect", schema: map[string]any{"$schema": "http://json-schema.org/draft-07/schema#"}, location: "/$schema"},
		{name: "nested $schema", schema: map[string]any{"items": map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema"}}, location: "/items/$schema"},
	}
	for _, tc := range forms {
		t.Run(tc.name, func(t *testing.T) {
			requireCompileError(t, mustJSON(t, tc.schema), streamschema.DefaultLimits(), streamschema.CompileUnsupported, tc.location)
		})
	}
}

// TestCompileAcceptsAnnotationsAndDialects guarantees the explicit
// annotation allowlist and supported dialects compile without effect.
func TestCompileAcceptsAnnotationsAndDialects(t *testing.T) {
	s := mustCompile(t, map[string]any{
		"$schema":      "https://json-schema.org/draft/2020-12/schema",
		"title":        "t",
		"description":  "d",
		"$comment":     "c",
		"format":       "uuid",
		"default":      map[string]any{"x": 1},
		"example":      1,
		"examples":     []any{1},
		"deprecated":   true,
		"readOnly":     true,
		"writeOnly":    false,
		"externalDocs": map[string]any{"url": "https://example.com"},
		"type":         "string",
	})
	require.NoError(t, validate(t, s, mustJSON(t, "not a uuid")))

	mustCompile(t, map[string]any{"$schema": "https://spec.openapis.org/oas/3.1/dialect/base"})
}

// TestCompileReportsInvalidSchemas guarantees malformed schemas are
// distinguished from unsupported ones, including JSON that a lenient parser
// would accept: duplicate keys, invalid UTF-8, and unpaired surrogates.
func TestCompileReportsInvalidSchemas(t *testing.T) {
	cases := []struct {
		name     string
		document string
		location string
	}{
		{name: "malformed JSON", document: `{"type":}`, location: "/type"},
		{name: "duplicate keys", document: `{"type":"string","type":"number"}`, location: ""},
		{name: "duplicate nested keys", document: `{"properties":{"a":true,"\u0061":false}}`, location: "/properties"},
		{name: "invalid UTF-8", document: "{\"title\":\"\xff\"}", location: "/title"},
		{name: "unpaired surrogate", document: `{"title":"\ud800"}`, location: "/title"},
		{name: "trailing data", document: `{} {}`, location: ""},
		{name: "root array", document: `[]`, location: ""},
		{name: "root number", document: `1`, location: ""},
		{name: "unknown type", document: `{"type":"strin"}`, location: "/type"},
		{name: "empty type list", document: `{"type":[]}`, location: "/type"},
		{name: "repeated type", document: `{"type":["string","string"]}`, location: "/type"},
		{name: "negative length", document: `{"maxLength":-1}`, location: "/maxLength"},
		{name: "fractional length", document: `{"maxLength":1.5}`, location: "/maxLength"},
		{name: "string length", document: `{"minItems":"1"}`, location: "/minItems"},
		{name: "required string", document: `{"required":"a"}`, location: "/required"},
		{name: "required repeated", document: `{"required":["a","a"]}`, location: "/required"},
		{name: "properties array", document: `{"properties":[]}`, location: "/properties"},
		{name: "property schema number", document: `{"properties":{"a":1}}`, location: "/properties/a"},
		{name: "string bound", document: `{"minimum":"1"}`, location: "/minimum"},
		{name: "boolean minimum", document: `{"minimum":true}`, location: "/minimum"},
		{name: "annotation of wrong type", document: `{"title":1}`, location: "/title"},
		{name: "$schema of wrong type", document: `{"$schema":1}`, location: "/$schema"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The bytes matter: these documents are malformed on purpose.
			requireCompileError(t, []byte(tc.document), streamschema.DefaultLimits(), streamschema.CompileInvalid, tc.location)
		})
	}
}

// TestCompileReportsFirstProblemDeterministically guarantees that a document
// with several problems always reports the same one: keywords are checked in
// sorted order, depth first.
func TestCompileReportsFirstProblemDeterministically(t *testing.T) {
	document := mustJSON(t, map[string]any{
		"pattern":    "a",
		"enum":       []any{1},
		"properties": map[string]any{"b": map[string]any{"const": 1}, "a": map[string]any{"type": "nope"}},
	})
	for range 20 {
		requireCompileError(t, document, streamschema.DefaultLimits(), streamschema.CompileUnsupported, "/enum")
	}
}

// TestCompileEnforcesSchemaLimits guarantees untrusted schemas, such as
// customer specs in frontline, cannot make compilation use unbounded memory.
func TestCompileEnforcesSchemaLimits(t *testing.T) {
	limits := streamschema.DefaultLimits()
	limits.SchemaBytesMax = 64
	requireCompileError(t, mustJSON(t, map[string]any{"description": strings.Repeat("x", 64)}), limits, streamschema.CompileLimit, "")

	limits = streamschema.DefaultLimits()
	limits.SchemaDepthMax = 4
	deep := map[string]any{"items": map[string]any{"items": map[string]any{"items": map[string]any{"items": map[string]any{}}}}}
	compileErr := requireCompileError(t, mustJSON(t, deep), limits, streamschema.CompileLimit, "/items/items/items/items")
	var parseErr *streamschema.ValidationError
	require.ErrorAs(t, compileErr.Err, &parseErr)
	require.Equal(t, streamschema.ReasonLimit, parseErr.Reason)

	limits = streamschema.DefaultLimits()
	limits.SchemaNodesMax = 3
	mustCompileWith(t, map[string]any{"items": map[string]any{"items": true}}, limits)
	requireCompileError(t, mustJSON(t, map[string]any{"items": map[string]any{"items": map[string]any{"items": true}}}), limits, streamschema.CompileLimit, "/items/items/items")

	// The bytes matter: the exponent is the subject.
	requireCompileError(t, []byte(`{"maximum":1e1000000001}`), streamschema.DefaultLimits(), streamschema.CompileLimit, "/maximum")
}

// TestCompileRejectsUnusableLimits guarantees that a zero or negative limit
// is a programming error rather than "unlimited".
func TestCompileRejectsUnusableLimits(t *testing.T) {
	limits := streamschema.DefaultLimits()
	limits.KeysRetainedBytesMax = 0
	compiled, err := streamschema.Compile([]byte(`true`), limits)
	require.Nil(t, compiled)
	require.Error(t, err)
	var compileErr *streamschema.CompileError
	require.NotErrorAs(t, err, &compileErr)
}
