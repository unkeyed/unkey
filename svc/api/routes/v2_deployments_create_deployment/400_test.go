package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v2_deployments_create_deployment"
)

func TestValidationErrors(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newUncalledRestate(t))
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

			res := testutil.CallRoute[map[string]any, openapi.BadRequestErrorResponse](h, route, headers, tc.body)
			require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, sent: %+v, received: %s", tc.body, res.RawBody)
			require.NotNil(t, res.Body)
			require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
			require.Equal(t, http.StatusBadRequest, res.Body.Error.Status)
			require.NotEmpty(t, res.Body.Meta.RequestId)
		})
	}
}

// TestInvalidEnvironmentSettings pins what a caller is told when the worker
// refuses an environment whose runtime or regional settings the deploy pipeline
// cannot satisfy. Which settings are wrong is the worker's to decide
// (deploy.TestCreateRejections); a refusal reaches the caller as an enum, so the
// message here names no field.
func TestInvalidEnvironmentSettings(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newRejectingRestate(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_ENVIRONMENT_NOT_DEPLOYABLE))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "nginx:latest")

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_environment_settings", res.Body.Error.Type)
}

// TestMalformedImageReference covers references the request schema accepts but a
// registry cannot serve. Parsing them belongs to the worker, which refuses
// before writing a row; this pins that the caller still gets a 400 rather than
// an id for a deployment that would only ever fail to pull.
func TestMalformedImageReference(t *testing.T) {
	h := testutil.NewHarness(t)
	route := newRoute(h, newRejectingRestate(t, hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_INVALID_IMAGE))
	h.Register(route)

	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	req := imageRequest(t, setup.Project.Slug, setup.App.Slug, setup.Environment.Slug, "Acme/Api:v1")

	res := testutil.CallRoute[handler.Request, openapi.BadRequestErrorResponse](h, route, authHeaders(setup.RootKey), req)
	require.Equal(t, http.StatusBadRequest, res.Status, "expected 400, received: %s", res.RawBody)
	require.Equal(t, "https://unkey.com/docs/errors/unkey/application/invalid_input", res.Body.Error.Type)
}
