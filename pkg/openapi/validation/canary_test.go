package validation

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// canary stands in for whatever a caller sent. It must never appear in a
// validation response, since that response is returned to the caller and written
// to the ClickHouse request log.
//
// This file is the invariant, and it is the file to read first. Everything else
// in the package can be rewritten as long as these tests keep passing. It is what
// sanitize_test.go was before messages were built from typed error kinds instead
// of by rewriting the library's prose: the assertions are re-aimed at the response
// rather than at regexes that no longer exist, but every leak that file pinned is
// still pinned here, end to end through the real validator.
//
// The four that were found by hand, each of which has its own test below:
//
//   - the Content-Type header, whose parameters carry arbitrary bytes
//   - the request path, which reached the caller through the summary line
//   - the JSON decoder's message, which names the character it choked on
//   - a limit of 1000 or more, which used to be dropped rather than reported
//
// describe_test.go covers the same ground one level down, over every error kind
// the JSON Schema library can produce.
const canary = "CANARYVALUE"

// probeSpec exercises the shapes that used to leak, plus the ones that could: a
// pattern, a length, a magnitude, an enum, an unknown property, a caller-named
// property inside a free-form object, and a header parameter, which is where
// frontline's customer specifications keep credentials.
const probeSpec = `
openapi: 3.1.0
info: {title: probe, version: 1.0.0}
paths:
  /probe:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [needed]
              properties:
                name: {type: string, pattern: "^[a-z]+$"}
                len: {type: string, maxLength: 3}
                big: {type: string, maxLength: 16384}
                num: {type: integer, maximum: 10}
                kind: {type: string, enum: [alpha, beta]}
                tags: {type: array, items: {type: string}, uniqueItems: true}
                needed: {type: string}
                free: {type: object, additionalProperties: {type: string}}
                strict:
                  type: object
                  propertyNames: {pattern: "^[a-z]+$"}
                  additionalProperties: {type: string}
              additionalProperties: false
      responses:
        "200": {description: ok}
  /params:
    get:
      parameters:
        - name: X-Secret
          in: header
          schema: {type: string, pattern: "^[a-z]+$"}
        - name: limit
          in: query
          schema: {type: integer, maximum: 10}
      responses:
        "200": {description: ok}
`

func TestValidateNeverEchoesRequestData(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	bodies := map[string]string{
		"pattern violation":         `{"needed":"x","name":"` + canary + `"}`,
		"length violation":          `{"needed":"x","len":"` + canary + `"}`,
		"type violation":            `{"needed":"x","num":"` + canary + `"}`,
		"magnitude violation":       `{"needed":"x","num":999999}`,
		"enum violation":            `{"needed":"x","kind":"` + canary + `"}`,
		"unknown property":          `{"needed":"x","` + canary + `":"x"}`,
		"missing required property": `{"name":"ok"}`,
		"duplicate items":           `{"needed":"x","tags":["` + canary + `","` + canary + `"]}`,
		"free-form object value":    `{"needed":"x","free":{"` + canary + `":1}}`,
		"rejected property name":    `{"needed":"x","strict":{"` + canary + `":"y"}}`,
		"malformed json":            `{"needed":"x","name":"` + canary,
		"several secrets at once":   `{"name":"` + canary + `","len":"` + canary + `","num":"` + canary + `"}`,
		"body is not an object":     `"` + canary + `"`,
	}

	for name, body := range bodies {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(body))
			require.NoError(t, err)
			r.Header.Set("Content-Type", "application/json")

			requireNoLeak(t, v.Validate(r), body)
		})
	}

	// frontline validates customer specifications, where a credential in a header
	// or a query parameter is normal.
	t.Run("header parameter", func(t *testing.T) {
		t.Parallel()

		r, err := http.NewRequest(http.MethodGet, "https://x/params", nil)
		require.NoError(t, err)
		r.Header.Set("X-Secret", canary)

		requireNoLeak(t, v.Validate(r), "header")
	})

	t.Run("query parameter", func(t *testing.T) {
		t.Parallel()

		r, err := http.NewRequest(http.MethodGet, "https://x/params?limit="+canary, nil)
		require.NoError(t, err)

		requireNoLeak(t, v.Validate(r), "query")
	})

	// And a URL path, which in a customer specification can be the credential.
	t.Run("path not in the specification", func(t *testing.T) {
		t.Parallel()

		r, err := http.NewRequest(http.MethodPost, "https://x/reset/"+canary, strings.NewReader("{}"))
		require.NoError(t, err)
		r.Header.Set("Content-Type", "application/json")

		requireNoLeak(t, v.Validate(r), "path")
	})
}

func requireNoLeak(t *testing.T, result *Result, input string) {
	t.Helper()

	require.NotNil(t, result, "should have failed validation: %s", input)
	require.NotContains(t, result.Detail, canary, "detail leaked request data")
	require.NotContains(t, result.Summary(), canary, "summary leaked request data")
	require.NotEmpty(t, result.Errors, "a rejected request must say something")

	for _, e := range result.Errors {
		require.NotContains(t, e.Message, canary, "message leaked request data")
		require.NotContains(t, e.Location, canary, "location leaked request data")
		require.NotEmpty(t, e.Location, "location is a required field of the response")
		require.NotEmpty(t, e.Message, "message is a required field of the response")
		if e.Fix != nil {
			require.NotContains(t, *e.Fix, canary, "fix leaked request data")
		}
	}
}

// Leak one. The Content-Type header is caller-controlled and the library
// interpolates it into both the summary line and the error message. Its
// parameters take arbitrary bytes, so `text/plain; boundary=<anything>` was a
// direct channel from a request header into the response and the request log.
func TestValidateNeverEchoesContentType(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	for _, contentType := range []string{
		"text/plain",
		"text/plain; boundary=" + canary,
		canary + "/" + canary,
	} {
		t.Run(contentType, func(t *testing.T) {
			t.Parallel()

			r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(`{"needed":"x"}`))
			require.NoError(t, err)
			r.Header.Set("Content-Type", contentType)

			result := v.Validate(r)
			require.NotNil(t, result)
			requireNoLeak(t, result, contentType)

			require.NotContains(t, result.Summary(), "text/plain")

			// Still useful: it names the problem and what the operation accepts.
			require.Equal(t,
				"POST operation for '/probe' does not accept the request content type",
				result.Detail)
			require.Equal(t, []ValidationError{{
				Location: "header.Content-Type",
				Message:  "is not a content type this operation accepts",
				Fix:      ptr("Send the request body using one of the content types this operation accepts: application/json."),
			}}, result.Errors)
		})
	}
}

// A parameter on a media type the operation does accept is not a failure at all,
// so nothing is echoed because nothing is reported. Pinned so the case above is
// understood as being about unsupported types specifically.
func TestValidateAcceptsSupportedTypeWithParameters(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(`{"needed":"x"}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json; charset="+canary)

	require.Nil(t, v.Validate(r))
}

// Leak two. The summary line is built from SpecPath, the matched path template,
// not from the request path. Our own paths have no parameters so the string is
// unchanged, but a customer specification behind frontline can put a credential
// in a path segment.
func TestValidateDetailUsesSpecPathNotRequestPath(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(`
openapi: 3.0.3
info: {title: probe, version: 1.0.0}
paths:
  /reset/{token}:
    post:
      parameters:
        - name: token
          in: path
          required: true
          schema: {type: string}
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                n: {type: integer}
      responses:
        "200": {description: ok}
`))
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/reset/sk_live_"+canary,
		strings.NewReader(`{"n":"not a number"}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	requireNoLeak(t, result, "path template")
	require.Equal(t, "POST request body for '/reset/{token}' failed to validate schema", result.Detail)
}

// Leak three. The JSON decoder names the character it choked on, so its message
// carries a byte the caller chose and cannot be forwarded. What goes out instead
// is written here.
func TestValidateNeverEchoesTheDecoderMessage(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	// The offending character is a single quote, which the decoder would quote back
	// as "invalid character '\''". Nothing about it reaches the response.
	r, err := http.NewRequest(http.MethodPost, "https://x/probe",
		strings.NewReader(`{'needed':'`+canary+`'}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	requireNoLeak(t, result, "malformed json")

	require.Equal(t, "POST request body for '/probe' failed to validate schema", result.Detail)
	require.Equal(t, []ValidationError{{
		Location: "body",
		Message:  "is not valid JSON",
		Fix:      ptr("Check the request body for a syntax error such as a missing quote, comma, or closing brace."),
	}}, result.Errors)
}

// Leak four. A limit of 1000 or more used to be dropped, because the shape being
// matched was the library's localized rendering ("maxLength: got 16,385, want
// 16,384") and the capture stopped at the thousands separator, so every large
// limit degraded to naming the keyword instead of reporting the bound.
//
// The limit now comes off kind.MaxLength.Want, which is the integer the schema
// declares, so it is reported in full and without the separator the library's
// printer added. A developer pastes the number back into a request, which the
// separator would break.
func TestValidateReportsLargeLimits(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/probe",
		strings.NewReader(`{"needed":"x","big":"`+strings.Repeat("a", 16385)+`"}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	require.NotNil(t, result)
	require.Equal(t, []ValidationError{
		{Location: "body.big", Message: "must be at most 16384 characters long", Fix: nil},
	}, result.Errors)

	// The length that was sent says how long the secret is, so it is absent too.
	require.NotContains(t, result.Summary(), "16385")
}

// A length, a magnitude, and a count are request data as well: they say how long
// the secret was. The library reports all three and this asserts none survive.
func TestValidateNeverEchoesSizes(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/probe",
		strings.NewReader(`{"needed":"x","len":"abcdefghijklm","num":424242}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	require.NotNil(t, result)

	rendered := result.Summary()
	// 13 is the length that was sent, 424242 the magnitude. 3 and 10 are the
	// limits from the specification and are expected to be there.
	require.NotContains(t, rendered, "13")
	require.NotContains(t, rendered, "424242")
	require.Contains(t, rendered, "must be at most 3 characters long")
	require.Contains(t, rendered, "must be at most 10")
}

func ptr(s string) *string { return &s }
