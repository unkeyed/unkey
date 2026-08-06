package openapi

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
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

func newTestExecutor(t *testing.T) *Executor {
	t.Helper()
	e, err := New(clock.New())
	require.NoError(t, err)
	return e
}

func TestExecute_EmptySpec(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	req := httptest.NewRequest("GET", "/anything", nil)

	//nolint:exhaustruct
	redactor, err := e.Execute(context.Background(), nil, req, &frontlinev1.OpenApiRequestValidation{})
	require.NoError(t, err)
	require.Nil(t, redactor)
}

func TestExecute_ValidRequest(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := `{"name":"alice"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := e.Execute(context.Background(), nil, req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.NoError(t, err)
}

func TestExecute_InvalidRequest(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	body := `{"wrong":"field"}`
	req := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err := e.Execute(context.Background(), nil, req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: minimalSpec,
	})
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.OpenApi.InvalidRequest.URN(), urn)
}

func TestExecute_InvalidSpec(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	req := httptest.NewRequest("GET", "/anything", nil)

	_, err := e.Execute(context.Background(), nil, req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: []byte("not valid yaml: [[["),
	})
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Frontline.Internal.InvalidConfiguration.URN(), urn)
}

func TestExecute_CachesCompiledValidator(t *testing.T) {
	t.Parallel()

	e := newTestExecutor(t)
	cfg := &frontlinev1.OpenApiRequestValidation{SpecYaml: minimalSpec}

	body := `{"name":"alice"}`
	req1 := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	_, err := e.Execute(context.Background(), nil, req1, cfg)
	require.NoError(t, err)

	req2 := httptest.NewRequest("POST", "/users", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	_, err = e.Execute(context.Background(), nil, req2, cfg)
	require.NoError(t, err)

	_, hit := e.cache.Get(context.Background(), string(minimalSpec))
	require.Equal(t, cache.Hit, hit)
}

func TestExecute_ReturnsSpecBodyRedactor(t *testing.T) {
	t.Parallel()

	spec := []byte(`
openapi: "3.0.0"
info:
  title: Test
  version: "1.0"
paths:
  /secrets:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                secret:
                  type: string
                  x-unkey-redact: true
                visible:
                  type: string
                  x-unkey-redact: false
      responses:
        "200":
          description: ok
`)
	body := `{"secret":"hide-me","visible":"keep-me"}`
	req := httptest.NewRequest("POST", "/secrets", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	redactor, err := newTestExecutor(t).Execute(context.Background(), nil, req, &frontlinev1.OpenApiRequestValidation{
		SpecYaml: spec,
	})
	require.NoError(t, err)
	require.NotNil(t, redactor)
	require.Equal(t, `{"secret":"[REDACTED]","visible":"keep-me"}`, string(redactor.Redact([]byte(body))))
}
