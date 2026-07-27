package validation

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// End to end through the validator: what a caller receives for a real request.

// The point of all this is still to tell a developer what to change.
func TestValidateStaysUseful(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/probe",
		strings.NewReader(`{"name":"NOPE","kind":"gamma"}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	require.NotNil(t, result)
	require.Equal(t, "POST request body for '/probe' failed to validate schema", result.Detail)

	require.Equal(t, []ValidationError{
		{Location: "body.kind", Message: "must be one of: 'alpha', 'beta'", Fix: nil},
		{Location: "body.name", Message: "must match the pattern '^[a-z]+$'", Fix: nil},
		{Location: "body.needed", Message: "is required", Fix: nil},
	}, result.Errors)

	require.Equal(t,
		"POST request body for '/probe' failed to validate schema: "+
			"body.kind must be one of: 'alpha', 'beta'; "+
			"body.name must match the pattern '^[a-z]+$'; "+
			"body.needed is required",
		result.Summary())
}

// Nested paths are what the location field is for.
func TestValidateReportsInstancePaths(t *testing.T) {
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
              properties:
                items:
                  type: array
                  items:
                    type: object
                    required: [name]
                    properties:
                      name: {type: string, pattern: "^[a-z]+$"}
                      tags: {type: array, items: {type: string}}
                free:
                  type: object
                  additionalProperties:
                    type: object
                    properties:
                      inner: {type: integer}
      responses:
        "200": {description: ok}
`)

	v, err := NewFromBytes(spec)
	require.NoError(t, err)

	body := `{"items":[{"name":"ok"},{"name":"BAD"},{"tags":[1]}],"free":{"` + canary + `":{"inner":"nope"}}}`
	r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(body))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	require.NotNil(t, result)

	locations := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		locations = append(locations, e.Location)
	}

	require.Equal(t, []string{
		// The caller named this one, so the segment is redacted and the declared
		// property below it survives.
		"body.free.*.inner",
		"body.items[1].name",
		"body.items[2].name",
		"body.items[2].tags[0]",
	}, locations)

	requireNoLeak(t, result, body)
}

// A header failure and a body failure in the same request are both reported, and
// the summary describes the first thing the validator looked at.
func TestValidateReportsParametersAndBodyTogether(t *testing.T) {
	t.Parallel()

	spec := []byte(`
openapi: 3.0.3
info: {title: probe, version: 1.0.0}
paths:
  /probe:
    post:
      parameters:
        - name: X-Version
          in: header
          required: true
          schema: {type: string, enum: [v1, v2]}
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: {type: string}
      responses:
        "200": {description: ok}
`)

	v, err := NewFromBytes(spec)
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(`{}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Version", canary)

	result := v.Validate(r)
	require.NotNil(t, result)
	require.Equal(t, "Header parameter 'X-Version' is not one of the values the schema allows", result.Detail)
	require.Equal(t, []ValidationError{
		{Location: "body.name", Message: "is required", Fix: nil},
		// The library checks a parameter's enum before the value reaches the schema,
		// so this comes from its own coarse phrase rather than from a constraint. Both
		// halves of that phrase are in a closed table, which is what makes it safe to
		// rebuild.
		{Location: "header.X-Version", Message: "is not one of the values the schema allows", Fix: nil},
	}, result.Errors)
}

// validate posts body to path and returns the result.
func validate(t *testing.T, v *Validator, path, body string) *Result {
	t.Helper()

	r, err := http.NewRequest(http.MethodPost, "https://x"+path, strings.NewReader(body))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	return v.Validate(r)
}
