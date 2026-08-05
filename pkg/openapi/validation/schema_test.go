package validation

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests for how the schema failure tree is walked: which node speaks for a
// failure, and which are suppressed because a node above said it better.

// A branching combinator reports the choice, not the branches. Told that a value
// must be a string and that it must be an integer, a caller learns less than one
// told it matched no allowed alternative, and for a 'oneOf' over required
// properties the branch failures read as though all of them were required.
func TestValidateReportsTheChoiceNotTheBranches(t *testing.T) {
	t.Parallel()

	spec := []byte(`
openapi: 3.0.3
info: {title: probe, version: 1.0.0}
paths:
  /exclusive:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                git: {type: string}
                image: {type: string}
              oneOf:
                - required: [git]
                - required: [image]
      responses:
        "200": {description: ok}
  /either:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                id:
                  anyOf:
                    - {type: string, minLength: 3}
                    - {type: integer}
      responses:
        "200": {description: ok}
  /excluded:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                name:
                  not: {type: string}
      responses:
        "200": {description: ok}
`)

	v, err := NewFromBytes(spec)
	require.NoError(t, err)

	tests := []struct {
		name string
		path string
		body string
		want []ValidationError
	}{
		{
			name: "oneOf matching nothing",
			path: "/exclusive",
			body: `{}`,
			want: []ValidationError{{
				Location: "body",
				Message:  "must match exactly one of the schemas defined for it, but it matched none",
				Fix:      nil,
			}},
		},
		{
			name: "oneOf matching more than one",
			path: "/exclusive",
			body: `{"git":"a","image":"b"}`,
			want: []ValidationError{{
				Location: "body",
				Message:  "must match exactly one of the schemas defined for it, but it matched more than one",
				Fix:      nil,
			}},
		},
		{
			name: "anyOf matching nothing",
			path: "/either",
			body: `{"id":{"secret":"` + canary + `"}}`,
			want: []ValidationError{{
				Location: "body.id",
				Message:  "must match at least one of the schemas defined for it",
				Fix:      nil,
			}},
		},
		{
			name: "not matching what it excludes",
			path: "/excluded",
			body: `{"name":"` + canary + `"}`,
			want: []ValidationError{{
				Location: "body.name",
				Message:  "must not match the schema defined under 'not'",
				Fix:      nil,
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := http.NewRequest(http.MethodPost, "https://x"+tt.path, strings.NewReader(tt.body))
			require.NoError(t, err)
			r.Header.Set("Content-Type", "application/json")

			result := v.Validate(r)
			requireNoLeak(t, result, tt.body)
			require.Equal(t, tt.want, result.Errors)
		})
	}
}

// A schema behind a $ref, or merged in through an allOf, is not a choice: there is
// one way to satisfy it and the child failure is the whole story. Those nodes are
// dropped so the specific constraint survives, which is the opposite of what
// happens to a branch.
func TestValidateKeepsFailuresUnderRefsAndAllOf(t *testing.T) {
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
                kind:
                  allOf:
                    - $ref: "#/components/schemas/Kind"
                  default: writeonly
                name:
                  $ref: "#/components/schemas/Name"
      responses:
        "200": {description: ok}
components:
  schemas:
    Kind: {type: string, enum: [recoverable, writeonly]}
    Name: {type: string, minLength: 3}
`)

	v, err := NewFromBytes(spec)
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/probe",
		strings.NewReader(`{"kind":"`+canary+`","name":"ab"}`))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	requireNoLeak(t, result, "refs")
	require.Equal(t, []ValidationError{
		{Location: "body.kind", Message: "must be one of: 'recoverable', 'writeonly'", Fix: nil},
		{Location: "body.name", Message: "must be at least 3 characters long", Fix: nil},
	}, result.Errors)
}

// `anyOf: [{$ref: X}, {type: "null"}]` is how OpenAPI 3.1 spells an optional
// object, and suppressing its branches would replace the actual problem with
// "must match at least one of the schemas defined for it". A choice with more than
// one surviving branch still gets the summary, because there the alternatives
// genuinely contradict each other.
func TestNullableChoiceReportsTheRealFailure(t *testing.T) {
	t.Parallel()

	spec := []byte(`
openapi: 3.1.0
info: {title: probe, version: 1.0.0}
paths:
  /nullable:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                healthcheck:
                  anyOf:
                    - type: object
                      required: [path]
                      properties:
                        path: {type: string, pattern: "^/"}
                    - type: "null"
      responses:
        "200": {description: ok}
  /choice:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                either:
                  anyOf:
                    - type: string
                      minLength: 5
                    - type: integer
                      minimum: 10
      responses:
        "200": {description: ok}
`)

	v, err := NewFromBytes(spec)
	require.NoError(t, err)

	t.Run("nullable reports the inner failure", func(t *testing.T) {
		t.Parallel()

		result := validate(t, v, "/nullable", `{"healthcheck":{"path":123}}`)
		require.NotNil(t, result)

		messages := map[string]string{}
		for _, e := range result.Errors {
			messages[e.Location] = e.Message
		}
		require.Contains(t, messages, "body.healthcheck.path")
		require.Equal(t, "must be a string, but a number was sent", messages["body.healthcheck.path"])
		require.NotContains(t, messages, "body.healthcheck")
	})

	t.Run("nullable still accepts null", func(t *testing.T) {
		t.Parallel()
		require.Nil(t, validate(t, v, "/nullable", `{"healthcheck":null}`))
	})

	t.Run("nullable reports a missing inner property", func(t *testing.T) {
		t.Parallel()

		result := validate(t, v, "/nullable", `{"healthcheck":{}}`)
		require.NotNil(t, result)
		require.Len(t, result.Errors, 1)
		require.Equal(t, "body.healthcheck.path", result.Errors[0].Location)
	})

	t.Run("a real choice keeps the summary", func(t *testing.T) {
		t.Parallel()

		result := validate(t, v, "/choice", `{"either":true}`)
		require.NotNil(t, result)
		require.Len(t, result.Errors, 1)
		require.Equal(t, "body.either", result.Errors[0].Location)
		require.Contains(t, result.Errors[0].Message, "must match at least one")
	})

	// The inner failures are rendered from typed kinds like everything else, so
	// widening what gets reported cannot widen what gets echoed.
	t.Run("nullable branch does not echo the value", func(t *testing.T) {
		t.Parallel()

		result := validate(t, v, "/nullable", `{"healthcheck":{"path":"`+canary+`"}}`)
		require.NotNil(t, result)
		require.NotContains(t, result.Summary(), canary)
		for _, e := range result.Errors {
			require.NotContains(t, e.Message, canary)
			require.NotContains(t, e.Location, canary)
		}
		require.Equal(t, `must match the pattern '^/'`, result.Errors[0].Message)
	})
}
