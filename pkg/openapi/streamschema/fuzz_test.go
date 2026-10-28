package streamschema_test

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"regexp"
	"strconv"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/fuzz"
	"github.com/unkeyed/unkey/pkg/openapi/streamschema"
)

// surrogateEscape matches a \u escape of a UTF-16 surrogate, paired or not.
var surrogateEscape = regexp.MustCompile(`\\u[dD][89a-fA-F]`)

// FuzzSyntaxAgreesWithEncodingJSON guarantees the strict parser accepts only
// JSON that encoding/json also accepts and reads the same way, rejects
// everything encoding/json rejects, and accepts all of the rest except the
// documented strict cases. The verdict must not depend on how the input is
// split across reads.
func FuzzSyntaxAgreesWithEncodingJSON(f *testing.F) {
	fuzz.Seed(f)
	s, err := streamschema.Compile([]byte(`true`), streamschema.DefaultLimits())
	require.NoError(f, err)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		var input bytes.Buffer
		writeFuzzJSON(c, &input, 0)
		chunk := int(c.Uint8()%16) + 1

		got := s.Validate(context.Background(), bytes.NewReader(input.Bytes()))
		requireSameOutcome(t, got, s.Validate(context.Background(), &chunkReader{data: input.Bytes(), size: chunk}), "chunked")

		reference := json.Valid(input.Bytes())
		if got == nil {
			require.True(t, reference, "accepted JSON that encoding/json rejects: %q", input.Bytes())
			require.True(t, utf8.Valid(input.Bytes()), "accepted invalid UTF-8: %q", input.Bytes())
			require.False(t, hasDuplicateNames(input.Bytes()), "accepted duplicate names: %q", input.Bytes())
			return
		}

		var failure *streamschema.ValidationError
		require.ErrorAs(t, got, &failure)
		require.NotEqual(t, streamschema.ReasonSchema, failure.Reason)
		strictOnly := !utf8.Valid(input.Bytes()) || surrogateEscape.Match(input.Bytes()) || hasDuplicateNames(input.Bytes())
		if reference && !strictOnly {
			t.Fatalf("rejected JSON that meets every strict rule: %q: %v", input.Bytes(), got)
		}
	})
}

// FuzzSchemaAgreesWithReference guarantees that for generated schemas in the
// supported subset and generated instances, the streaming verdict matches an
// independent in-memory evaluator that compares numbers with math/big.
func FuzzSchemaAgreesWithReference(f *testing.F) {
	fuzz.Seed(f)

	f.Fuzz(func(t *testing.T, data []byte) {
		c := fuzz.New(t, data)
		document := mustJSON(t, fuzzSchema(c, 0))
		input := mustJSON(t, fuzzInstance(c, 0))
		chunk := int(c.Uint8()%16) + 1

		s, err := streamschema.Compile(document, streamschema.DefaultLimits())
		require.NoError(t, err, "%s", document)
		got := s.Validate(context.Background(), bytes.NewReader(input))
		requireSameOutcome(t, got, s.Validate(context.Background(), &chunkReader{data: input, size: chunk}), "chunked")

		if referenceValid(decodeNumbers(t, document), decodeNumbers(t, input)) {
			require.NoError(t, got, "schema %s, instance %s", document, input)
			return
		}
		var failure *streamschema.ValidationError
		require.ErrorAs(t, got, &failure, "schema %s, instance %s", document, input)
		require.Equal(t, streamschema.ReasonSchema, failure.Reason, failure.Error())
	})
}

// fuzzStringPieces are string contents covering escapes, multi-byte UTF-8,
// and the invalid forms the strict parser must reject.
var fuzzStringPieces = []string{
	"a", "é", "😀", "\x7f", `\n`, `\"`, `\\`, `\/`, `\u0041`, `\u00e9`,
	`\ud83d\ude00`, `\ud800`, `\udc00`, `\ud800\u0041`, `\x`, `\u12`,
	"\x01", "\xff", "\xc0\xaf", "\xed\xa0\x80", `"`, `\`,
}

// fuzzNumberPieces concatenate into valid and invalid number tokens.
var fuzzNumberPieces = []string{"0", "1", "12", "-", ".", "5", "e", "E", "+", "00", "e-3"}

// fuzzNames are member names, including an escaped duplicate of "a".
var fuzzNames = []string{`"a"`, `"b"`, `"\u0061"`, `""`, `"a/b"`}

// writeFuzzJSON writes a mostly well-formed JSON value with fuzzer-chosen
// mutations.
func writeFuzzJSON(c *fuzz.Consumer, b *bytes.Buffer, depth int) {
	kind := c.Uint8() % 12
	if depth >= 5 && kind >= 4 && kind <= 7 {
		kind = 0
	}
	switch kind {
	case 0:
		b.WriteString([]string{"null", "true", "false", "nul", "truex"}[c.Uint8()%5])
	case 1, 2:
		for range c.Uint8()%4 + 1 {
			b.WriteString(fuzzNumberPieces[int(c.Uint8())%len(fuzzNumberPieces)])
		}
	case 3:
		b.WriteByte('"')
		for range c.Uint8() % 6 {
			b.WriteString(fuzzStringPieces[int(c.Uint8())%len(fuzzStringPieces)])
		}
		if c.Uint8()%16 != 0 {
			b.WriteByte('"')
		}
	case 4, 5:
		b.WriteByte('[')
		for i := range int(c.Uint8() % 4) {
			if i > 0 || c.Uint8()%16 == 0 {
				b.WriteByte(',')
			}
			writeFuzzJSON(c, b, depth+1)
		}
		b.WriteByte(']')
	case 6, 7:
		b.WriteByte('{')
		for i := range int(c.Uint8() % 4) {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(fuzzNames[int(c.Uint8())%len(fuzzNames)])
			b.WriteByte(':')
			writeFuzzJSON(c, b, depth+1)
		}
		b.WriteByte('}')
	case 8:
		b.WriteString([]string{" ", "\t", "\n", "\r", "\v", "\xef\xbb\xbf"}[c.Uint8()%6])
		writeFuzzJSON(c, b, depth)
	default:
		b.Write(c.BytesN(int(c.Uint8() % 3)))
	}
}

// hasDuplicateNames reports whether a document encoding/json accepts has an
// object with two equal decoded member names. It returns false for documents
// encoding/json rejects.
func hasDuplicateNames(data []byte) bool {
	type frame struct {
		names     map[string]bool
		object    bool
		expectKey bool
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var stack []frame
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		if top := len(stack) - 1; top >= 0 && stack[top].object && stack[top].expectKey {
			if name, ok := token.(string); ok {
				if stack[top].names[name] {
					return true
				}
				stack[top].names[name] = true
				stack[top].expectKey = false
				continue
			}
		}
		switch token {
		case json.Delim('{'):
			stack = append(stack, frame{names: map[string]bool{}, object: true, expectKey: true})
			continue
		case json.Delim('['):
			stack = append(stack, frame{names: nil, object: false, expectKey: false})
			continue
		case json.Delim('}'), json.Delim(']'):
			stack = stack[:len(stack)-1]
		}
		if top := len(stack) - 1; top >= 0 && stack[top].object {
			stack[top].expectKey = true
		}
	}
}

// fuzzTypeNames are the JSON Schema type names.
var fuzzTypeNames = []string{"null", "boolean", "object", "array", "number", "integer", "string"}

// fuzzMemberNames are the names generated schemas and instances share, so
// properties and required often apply.
var fuzzMemberNames = []string{"a", "b", "c"}

// fuzzSchema generates a schema in the supported subset.
func fuzzSchema(c *fuzz.Consumer, depth int) any {
	if depth >= 3 || c.Uint8()%6 == 0 {
		return c.Bool()
	}
	schema := map[string]any{}
	flags := c.Uint16()
	if flags&1 != 0 {
		first := fuzzTypeNames[int(c.Uint8())%len(fuzzTypeNames)]
		second := fuzzTypeNames[int(c.Uint8())%len(fuzzTypeNames)]
		if c.Bool() || first == second {
			schema["type"] = first
		} else {
			schema["type"] = []string{first, second}
		}
	}
	counts := []string{"minLength", "maxLength", "minItems", "maxItems"}
	for i, name := range counts {
		if flags&(2<<i) != 0 {
			schema[name] = int(c.Uint8() % 6)
		}
	}
	bounds := []string{"minimum", "maximum", "exclusiveMinimum", "exclusiveMaximum"}
	for i, name := range bounds {
		if flags&(32<<i) != 0 {
			schema[name] = fuzzNumber(c)
		}
	}
	if flags&512 != 0 {
		properties := map[string]any{}
		for range c.Uint8() % 3 {
			properties[fuzzMemberNames[int(c.Uint8())%len(fuzzMemberNames)]] = fuzzSchema(c, depth+1)
		}
		schema["properties"] = properties
	}
	if flags&1024 != 0 {
		required := []string{}
		for _, name := range fuzzMemberNames {
			if c.Bool() {
				required = append(required, name)
			}
		}
		schema["required"] = required
	}
	if flags&2048 != 0 {
		schema["additionalProperties"] = fuzzSchema(c, depth+1)
	}
	if flags&4096 != 0 {
		schema["items"] = fuzzSchema(c, depth+1)
	}
	return schema
}

// fuzzInstance generates an arbitrary JSON value.
func fuzzInstance(c *fuzz.Consumer, depth int) any {
	kind := c.Uint8() % 7
	if depth >= 4 && kind >= 5 {
		kind = 0
	}
	switch kind {
	case 0:
		return nil
	case 1:
		return c.Bool()
	case 2, 3:
		return fuzzNumber(c)
	case 4:
		return string(c.BytesN(int(c.Uint8() % 8)))
	case 5:
		items := []any{}
		for range c.Uint8() % 5 {
			items = append(items, fuzzInstance(c, depth+1))
		}
		return items
	default:
		members := map[string]any{}
		for range c.Uint8() % 4 {
			members[fuzzMemberNames[int(c.Uint8())%len(fuzzMemberNames)]] = fuzzInstance(c, depth+1)
		}
		return members
	}
}

// fuzzNumber generates a valid JSON number with an optional sign, fraction,
// and exponent.
func fuzzNumber(c *fuzz.Consumer) json.Number {
	var b []byte
	if c.Bool() {
		b = append(b, '-')
	}
	b = strconv.AppendUint(b, uint64(c.Uint16()%300), 10)
	if c.Bool() {
		b = append(b, '.')
		b = strconv.AppendUint(b, uint64(c.Uint8()%100), 10)
	}
	if c.Bool() {
		b = append(b, 'e')
		b = strconv.AppendInt(b, int64(c.Int8()%20), 10)
	}
	return json.Number(b)
}

// decodeNumbers decodes data keeping numbers as json.Number.
func decodeNumbers(t *testing.T, data []byte) any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	require.NoError(t, decoder.Decode(&value))
	return value
}

// referenceValid evaluates the supported subset over decoded values. It is
// written independently of the package, with math/big for numbers, to serve as
// a differential oracle.
func referenceValid(schema any, instance any) bool {
	if accept, ok := schema.(bool); ok {
		return accept
	}
	s := schema.(map[string]any)
	if types, ok := s["type"]; ok && !referenceTypeMatches(types, instance) {
		return false
	}

	switch value := instance.(type) {
	case string:
		length := int64(utf8.RuneCountInString(value))
		return referenceAtLeast(s, "minLength", length) && referenceAtMost(s, "maxLength", length)
	case json.Number:
		n := referenceRat(value)
		checks := map[string]func(int) bool{
			"minimum":          func(c int) bool { return c >= 0 },
			"maximum":          func(c int) bool { return c <= 0 },
			"exclusiveMinimum": func(c int) bool { return c > 0 },
			"exclusiveMaximum": func(c int) bool { return c < 0 },
		}
		for name, ok := range checks {
			if bound, present := s[name]; present && !ok(n.Cmp(referenceRat(bound.(json.Number)))) {
				return false
			}
		}
		return true
	case []any:
		length := int64(len(value))
		if !referenceAtLeast(s, "minItems", length) || !referenceAtMost(s, "maxItems", length) {
			return false
		}
		if items, ok := s["items"]; ok {
			for _, item := range value {
				if !referenceValid(items, item) {
					return false
				}
			}
		}
		return true
	case map[string]any:
		properties, _ := s["properties"].(map[string]any)
		for name, member := range value {
			child, named := properties[name]
			if !named {
				child, named = s["additionalProperties"]
			}
			if named && !referenceValid(child, member) {
				return false
			}
		}
		required, _ := s["required"].([]any)
		for _, name := range required {
			if _, ok := value[name.(string)]; !ok {
				return false
			}
		}
		return true
	default:
		return true
	}
}

// referenceTypeMatches reports whether instance has one of the type names.
func referenceTypeMatches(types any, instance any) bool {
	names, ok := types.([]any)
	if !ok {
		names = []any{types}
	}
	for _, name := range names {
		switch value := instance.(type) {
		case nil:
			ok = name == "null"
		case bool:
			ok = name == "boolean"
		case string:
			ok = name == "string"
		case []any:
			ok = name == "array"
		case map[string]any:
			ok = name == "object"
		case json.Number:
			ok = name == "number" || (name == "integer" && referenceRat(value).IsInt())
		}
		if ok {
			return true
		}
	}
	return false
}

// referenceAtLeast reports whether value meets the count keyword name.
func referenceAtLeast(s map[string]any, name string, value int64) bool {
	bound, ok := s[name]
	return !ok || value >= referenceRat(bound.(json.Number)).Num().Int64()
}

// referenceAtMost reports whether value meets the count keyword name.
func referenceAtMost(s map[string]any, name string, value int64) bool {
	bound, ok := s[name]
	return !ok || value <= referenceRat(bound.(json.Number)).Num().Int64()
}

// referenceRat parses an exact rational.
func referenceRat(number json.Number) *big.Rat {
	r, ok := new(big.Rat).SetString(string(number))
	if !ok {
		panic("invalid number " + string(number))
	}
	return r
}
