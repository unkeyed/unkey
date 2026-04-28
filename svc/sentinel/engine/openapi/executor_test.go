package openapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	sentinelv1 "github.com/unkeyed/unkey/gen/proto/sentinel/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

var minimalSpec = []byte(`
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name:
                  type: string
      responses:
        "200":
          description: ok
`)

func TestExecute_EmptySpec(t *testing.T) {
	t.Parallel()

	e := New()
	req := httptest.NewRequest("GET", "/anything", nil)

	//nolint:exhaustruct
	err := e.Execute(context.Background(), nil, req, &sentinelv1.OpenApiRequestValidation{})
	require.NoError(t, err)
}

func TestExecute_ValidRequest(t *testing.T) {
	t.Parallel()

	e := New()
	body := `{"name":"alice"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	err := e.Execute(context.Background(), nil, req, &sentinelv1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.NoError(t, err)
}

func TestExecute_InvalidRequest(t *testing.T) {
	t.Parallel()

	e := New()
	body := `{"wrong":"field"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	err := e.Execute(context.Background(), nil, req, &sentinelv1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Sentinel.OpenApi.InvalidRequest.URN(), urn)
}

func TestExecute_InvalidSpec(t *testing.T) {
	t.Parallel()

	e := New()
	req := httptest.NewRequest("GET", "/anything", nil)

	err := e.Execute(context.Background(), nil, req, &sentinelv1.OpenApiRequestValidation{
		SpecYaml: []byte("not valid yaml: [[["),
	})
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Sentinel.Internal.InvalidConfiguration.URN(), urn)
}

func TestExecute_CachesCompiledValidator(t *testing.T) {
	t.Parallel()

	e := New()
	cfg := &sentinelv1.OpenApiRequestValidation{SpecYaml: minimalSpec}

	body := `{"name":"alice"}`
	req1 := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	require.NoError(t, e.Execute(context.Background(), nil, req1, cfg))

	req2 := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	require.NoError(t, e.Execute(context.Background(), nil, req2, cfg))

	count := 0
	e.validators.Range(func(_, _ any) bool {
		count++
		return true
	})
	require.Equal(t, 1, count)
}
