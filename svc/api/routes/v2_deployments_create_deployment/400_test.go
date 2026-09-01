package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
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
		{"image uppercase name", body(map[string]any{"image": map[string]any{"dockerImage": "Acme/Api:v1"}})},
		{"image empty tag", body(map[string]any{"image": map[string]any{"dockerImage": "acme/api:"}})},
		{"image truncated digest", body(map[string]any{"image": map[string]any{"dockerImage": "acme/api@sha256:abc123"}})},
		{"image over the column width", body(map[string]any{"image": map[string]any{"dockerImage": "ghcr.io/acme/" + strings.Repeat("a", 244)}})},
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

// TestInvalidEnvironmentSettings covers the create-time pre-flight that rejects an
// environment whose runtime or regional settings would fail the deploy pipeline,
// so the caller gets a synchronous 400 instead of a build that dies mid-flight.
func TestInvalidEnvironmentSettings(t *testing.T) {
	const docsURL = "https://unkey.com/docs/errors/unkey/application/invalid_environment_settings"

	t.Run("no schedulable region", func(t *testing.T) {
		h := testutil.NewHarness(t)
		capture := &ctrlCapture{}
		route := newRoute(h, capture)
		h.Register(route)

		// The seeder gives sane runtime settings but never configures a region.
		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			Permissions: []string{"environment.*.create_deployment"},
		})

		req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		require.Equal(t, docsURL, res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "no schedulable regions")
		require.False(t, capture.called, "ctrl must not be called for an undeployable environment")
	})

	t.Run("invalid runtime bounds reports every field", func(t *testing.T) {
		h := testutil.NewHarness(t)
		capture := &ctrlCapture{}
		route := newRoute(h, capture)
		h.Register(route)

		setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
			Permissions: []string{"environment.*.create_deployment"},
		})
		// A region is configured, so only the zeroed runtime bounds should fail.
		seedDeployableRegion(t, h, setup)
		zeroRuntimeSettings(t, h, setup)

		req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

		res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, authHeaders(setup.RootKey), req)
		require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
		require.Equal(t, docsURL, res.Body.Error.Type)
		require.Contains(t, res.Body.Error.Detail, "Port must be between 1 and 65535")
		require.Contains(t, res.Body.Error.Detail, "CPU millicores must be at least 250")
		require.Contains(t, res.Body.Error.Detail, "MemoryMib must be at least 256")
		require.NotContains(t, res.Body.Error.Detail, "region", "a configured region must not be reported")
		require.False(t, capture.called, "ctrl must not be called for an undeployable environment")
	})
}
