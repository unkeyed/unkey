package validation

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Tests for the shape of the result: ordering, bounds, and the summary line.

// Several independent problems in one request must all be reported, in a stable
// order. The JSON Schema library ranges over a map to check an object's
// properties, so the order it reports them in changes between runs.
func TestValidateReportsEveryProblemDeterministically(t *testing.T) {
	t.Parallel()

	v, err := NewFromBytes([]byte(probeSpec))
	require.NoError(t, err)

	body := `{"name":"BAD","len":"toolong","num":99,"kind":"gamma"}`

	var first []ValidationError
	for range 20 {
		r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(body))
		require.NoError(t, err)
		r.Header.Set("Content-Type", "application/json")

		result := v.Validate(r)
		require.NotNil(t, result)
		require.Len(t, result.Errors, 5)

		if first == nil {
			first = result.Errors

			continue
		}
		require.Equal(t, first, result.Errors)
	}

	require.Equal(t, []string{
		"body.kind", "body.len", "body.name", "body.needed", "body.num",
	}, []string{
		first[0].Location, first[1].Location, first[2].Location,
		first[3].Location, first[4].Location,
	})
}

// A body that violates a constraint on every one of many properties must not turn
// into an unbounded response, or an unbounded row in the request log.
func TestValidateBoundsTheErrorList(t *testing.T) {
	t.Parallel()

	var schema strings.Builder
	schema.WriteString(`
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
`)
	var body strings.Builder
	body.WriteString("{")
	for i := range 60 {
		name := "p" + string(rune('a'+i%26)) + string(rune('a'+i/26))
		schema.WriteString("                " + name + ": {type: integer}\n")
		if i > 0 {
			body.WriteString(",")
		}
		body.WriteString(`"` + name + `":"` + canary + `"`)
	}
	body.WriteString("}")
	schema.WriteString(`      responses:
        "200": {description: ok}
`)

	v, err := NewFromBytes([]byte(schema.String()))
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(body.String()))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	require.NotNil(t, result)
	require.Len(t, result.Errors, maxErrors+1)
	require.Equal(t, ValidationError{
		Location: "request",
		Message:  "has further validation errors that were not reported",
		Fix:      nil,
	}, result.Errors[maxErrors])

	// The summary is bounded separately, since it is one line.
	require.Equal(t, maxSummaryErrors, strings.Count(result.Summary(), "; "))
	require.True(t, strings.HasSuffix(result.Summary(), "; …"))

	requireNoLeak(t, result, body.String())
}

// A rejected request that produced no entry at all would be a response with an
// empty errors list, which the published schema does not allow for.
func TestValidateAlwaysReportsSomething(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		[]ValidationError{{Location: "request", Message: fallbackMessage, Fix: nil}},
		normalize(nil))
}

func TestSummary(t *testing.T) {
	t.Parallel()

	var absent *Result
	require.Empty(t, absent.Summary())

	require.Equal(t, "One or more fields failed validation",
		(&Result{Detail: genericDetail, Errors: nil}).Summary())

	require.Equal(t, "detail: body.a is required",
		(&Result{
			Detail: "detail",
			Errors: []ValidationError{{Location: "body.a", Message: "is required", Fix: nil}},
		}).Summary())
}
