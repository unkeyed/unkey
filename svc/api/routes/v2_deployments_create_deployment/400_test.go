package handler_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	capture := &ctrlCapture{}
	route := newRoute(h, capture)
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	headers := authHeaders(setup.RootKey)

	// body merges the shared identifiers with the source object under test.
	// Invalid combinations are sent as raw JSON so the schema and handler
	// validation can be exercised directly.
	body := func(fields map[string]any) map[string]any {
		m := map[string]any{
			"project":     setup.Project.Slug,
			"app":         setup.App.Slug,
			"environment": setup.Environment.Slug,
		}
		for k, v := range fields {
			m[k] = v
		}
		return m
	}

	cases := []struct {
		name string
		body map[string]any
	}{
		{"image missing dockerImage", body(map[string]any{"image": map[string]any{}})},
		{"image whitespace dockerImage", body(map[string]any{"image": map[string]any{"dockerImage": "   "}})},
		{"git fork without commitSha", body(map[string]any{"git": map[string]any{"repository": "contributor/acme-api"}})},
		{"git fork bad charset", body(map[string]any{"git": map[string]any{"commitSha": "abc123", "repository": "bad repo!"}})},
		{"git fork path traversal", body(map[string]any{"git": map[string]any{"commitSha": "abc123", "repository": "../../etc/passwd"}})},
		{"deployment missing deploymentId", body(map[string]any{"deployment": map[string]any{}})},
		{"no source", body(map[string]any{})},
		{"multiple sources", body(map[string]any{"image": map[string]any{"dockerImage": "nginx:latest"}, "git": map[string]any{"branch": "main"}})},
		{"missing project", map[string]any{
			"app":         setup.App.Slug,
			"environment": setup.Environment.Slug,
			"image":       map[string]any{"dockerImage": "nginx:latest"},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			capture.called = false

			res := testutil.CallRoute[map[string]any, openapi.BadRequestErrorResponse](h, route, headers, tc.body)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.body, res.RawBody)
			require.NotNil(t, res.Body)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
			require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
			require.NotEmpty(t, res.Body.Meta.RequestId)
			require.False(t, capture.called, "ctrl must not be called on a validation failure")
		})
	}
}
