package redaction

import (
	"encoding/json"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

// Tests for the redaction contract itself: what comes out, and what it costs.

// secret is the canary every test looks for. If it survives a Redact call on a
// body where it sits in a redacted member, the redactor has failed.
//
// Deliberately not shaped like any real credential: a fixture that looks like a
// live key trips secret scanners on push, and this file is full of them.
const secret = "FIXTURE-not-a-real-credential-01234567890123456789"

// The paths under test mirror the shapes the real spec produces: secrets at the
// root of a request body, the same secrets nested under data in a response, and
// values inside an array of objects.
func testRedactor() *Redactor {
	return New([]string{
		"key",
		"plaintext",
		"value",
		"token",
		"sessionId",
		"data.key",
		"data.plaintext",
		"variables[].key",
		"variables[].value",
		"data.variables[].value",
		"[].key",
	})
}

func TestRedact(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "string value",
			in:   `{"key":"` + secret + `"}`,
			want: `{"key":"[REDACTED]"}`,
		},
		{
			name: "whitespace around the colon is preserved",
			in:   `{"key"  :  "` + secret + `"}`,
			want: `{"key"  :  "[REDACTED]"}`,
		},
		{
			name: "tabs and newlines around the colon",
			in:   "{\"key\"\t:\n\"" + secret + "\"}",
			want: "{\"key\"\t:\n\"[REDACTED]\"}",
		},
		{
			name: "escaped quotes inside the value",
			in:   `{"key":"a\"b\\c","name":"visible"}`,
			want: `{"key":"[REDACTED]","name":"visible"}`,
		},
		{
			name: "trailing backslash pair does not swallow the value",
			in:   `{"key":"a\\","name":"visible"}`,
			want: `{"key":"[REDACTED]","name":"visible"}`,
		},
		{
			name: "numeric value",
			in:   `{"value":12345,"name":"visible"}`,
			want: `{"value":"[REDACTED]","name":"visible"}`,
		},
		{
			name: "literal null",
			in:   `{"value":null}`,
			want: `{"value":"[REDACTED]"}`,
		},
		{
			name: "boolean value",
			in:   `{"value":true,"other":false}`,
			want: `{"value":"[REDACTED]","other":false}`,
		},
		{
			name: "object value is replaced wholesale",
			in:   `{"value":{"nested":"` + secret + `"},"name":"visible"}`,
			want: `{"value":"[REDACTED]","name":"visible"}`,
		},
		{
			name: "array value is replaced wholesale",
			in:   `{"value":["` + secret + `",{"a":[1,2]}],"name":"visible"}`,
			want: `{"value":"[REDACTED]","name":"visible"}`,
		},
		{
			name: "object value containing braces inside strings",
			in:   `{"value":{"a":"}{"},"name":"visible"}`,
			want: `{"value":"[REDACTED]","name":"visible"}`,
		},
		{
			name: "nested occurrence matching data.key",
			in:   `{"data":{"key":"` + secret + `","keyId":"key_123"},"meta":{"requestId":"req_1"}}`,
			want: `{"data":{"key":"[REDACTED]","keyId":"key_123"},"meta":{"requestId":"req_1"}}`,
		},
		{
			name: "repeated occurrences in an array",
			in:   `[{"key":"a"},{"key":"b"},{"key":"c"}]`,
			want: `[{"key":"[REDACTED]"},{"key":"[REDACTED]"},{"key":"[REDACTED]"}]`,
		},
		{
			name: "multiple distinct fields",
			in:   `{"key":"a","plaintext":"b","name":"visible","token":"c"}`,
			want: `{"key":"[REDACTED]","plaintext":"[REDACTED]","name":"visible","token":"[REDACTED]"}`,
		},
		{
			name: "env var payload redacts inside the array",
			in:   `{"variables":[{"key":"DATABASE_URL","value":"` + secret + `","kind":"recoverable"}]}`,
			want: `{"variables":[{"key":"[REDACTED]","value":"[REDACTED]","kind":"recoverable"}]}`,
		},
		{
			name: "matching is exact, not a prefix or suffix",
			in:   `{"keyId":"key_123","apiKey":"visible","key_":"visible","valueOf":"visible"}`,
			want: `{"keyId":"key_123","apiKey":"visible","key_":"visible","valueOf":"visible"}`,
		},
		{
			name: "the same name in value position is left alone",
			in:   `{"name":"key","tags":["key","value"]}`,
			want: `{"name":"key","tags":["key","value"]}`,
		},
		{
			name: "already redacted input is unchanged",
			in:   `{"key":"[REDACTED]"}`,
			want: `{"key":"[REDACTED]"}`,
		},
		{
			name: "empty string value",
			in:   `{"key":""}`,
			want: `{"key":"[REDACTED]"}`,
		},
		{
			name: "unicode escapes in the value",
			in:   `{"key":"sk_\u00e9\u00e8"}`,
			want: `{"key":"[REDACTED]"}`,
		},
	}

	r := testRedactor()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := string(r.Redact([]byte(tt.in)))
			require.Equal(t, tt.want, got)
			require.NotContains(t, got, secret)
		})
	}
}

func TestRedact_OutputStaysValidJSON(t *testing.T) {
	t.Parallel()

	bodies := []string{
		`{"key":"secret","n":1,"b":true,"arr":[1,2,3],"obj":{"a":"b"}}`,
		`{"value":{"deep":{"deeper":[1,2,{"key":"secret"}]}}}`,
		`[{"token":"a"},{"sessionId":"b"}]`,
		`{"value":12345}`,
		`{"value":null}`,
	}

	r := testRedactor()
	for _, body := range bodies {
		require.True(t, json.Valid([]byte(body)), "test input must be valid JSON: %s", body)
		got := r.Redact([]byte(body))
		require.True(t, json.Valid(got), "redacted output is not valid JSON: %s", got)
	}
}

func TestRedact_PassesThroughUntouched(t *testing.T) {
	t.Parallel()

	inputs := []string{
		``,
		`{}`,
		`{"valid":true,"code":"VALID","keyId":"key_123"}`,
		`{"error":{"title":"Bad Request","status":400},"meta":{"requestId":"req_1"}}`,
		`this is plain text mentioning key and value and plaintext`,
		"\x00\x01binary\xff",
	}

	r := testRedactor()
	for _, in := range inputs {
		require.Equal(t, in, string(r.Redact([]byte(in))))
	}
}

func TestRedact_DoesNotMutateInput(t *testing.T) {
	t.Parallel()

	in := []byte(`{"key":"` + secret + `"}`)
	original := string(in)

	got := testRedactor().Redact(in)

	require.Equal(t, original, string(in), "input slice was mutated")
	require.NotEqual(t, original, string(got))
}

func TestRedact_NoFieldsIsPassThrough(t *testing.T) {
	t.Parallel()

	in := `{"key":"` + secret + `"}`
	require.Equal(t, in, string(New(nil).Redact([]byte(in))))

	var nilRedactor *Redactor
	require.Equal(t, in, string(nilRedactor.Redact([]byte(in))))
}

func TestRedact_Idempotent(t *testing.T) {
	t.Parallel()

	r := testRedactor()
	in := []byte(`{"key":"` + secret + `","name":"visible"}`)

	once := r.Redact(in)
	twice := r.Redact(once)

	require.Equal(t, string(once), string(twice))
}

func TestRedactString(t *testing.T) {
	t.Parallel()

	r := testRedactor()

	require.Equal(t, `{"key":"[REDACTED]"}`, r.RedactString([]byte(`{"key":"`+secret+`"}`)))
	require.Equal(t, `{"keyId":"key_123"}`, r.RedactString([]byte(`{"keyId":"key_123"}`)))
	require.Equal(t, "", r.RedactString(nil))
	require.Equal(t, "", r.RedactString([]byte{}))
}

// The whole point of RedactString is skipping the copy, so assert the aliasing
// rather than trusting it: an untouched body must hand back the same memory, and
// a redacted one must not, since the caller would otherwise be holding a string
// over a buffer the redactor rewrote.
func TestRedactString_AliasesOnlyWhenUntouched(t *testing.T) {
	t.Parallel()

	r := testRedactor()

	// Compare addresses as integers. testify's Equal on two *uint8 would compare
	// the bytes they point at, and every JSON body starts with the same one.
	address := func(b []byte) uintptr { return uintptr(unsafe.Pointer(unsafe.SliceData(b))) }
	stringAddress := func(s string) uintptr { return uintptr(unsafe.Pointer(unsafe.StringData(s))) }

	untouched := []byte(`{"keyId":"key_123","name":"visible"}`)
	got := r.RedactString(untouched)
	require.Equal(t, address(untouched), stringAddress(got), "an untouched body should not be copied")

	redacted := []byte(`{"key":"` + secret + `"}`)
	got = r.RedactString(redacted)
	require.NotEqual(t, address(redacted), stringAddress(got), "a redacted body must not alias the input")
	require.Equal(t, `{"key":"`+secret+`"}`, string(redacted), "input was mutated")
}
