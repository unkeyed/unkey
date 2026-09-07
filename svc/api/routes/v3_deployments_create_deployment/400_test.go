package handler_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

func TestRequestValidation(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	route := &handler.Handler{DB: h.DB, CtrlClient: &testutil.MockDeploymentClient{}}
	h.Register(route)

	base := map[string]any{
		"project":     setup.Project.Slug,
		"app":         setup.App.Slug,
		"environment": setup.Environment.Slug,
	}

	for _, tc := range []struct {
		name   string
		source map[string]any
	}{
		{name: "missing OCI image", source: map[string]any{"oci": map[string]any{}}},
		{name: "whitespace OCI image", source: map[string]any{"oci": map[string]any{"image": "   "}}},
		{name: "OCI image too long", source: map[string]any{"oci": map[string]any{"image": strings.Repeat("a", 253) + ":tag"}}},
		{name: "unknown OCI field", source: map[string]any{"oci": map[string]any{"image": "nginx:latest", "unknown": true}}},
		{name: "OCI and git sources", source: map[string]any{"oci": map[string]any{"image": "nginx"}, "git": map[string]any{"branch": "main"}}},
		{name: "OCI and deployment sources", source: map[string]any{"oci": map[string]any{"image": "nginx"}, "deployment": map[string]any{"deploymentId": "d_previous"}}},
		{name: "git and deployment sources", source: map[string]any{"git": map[string]any{"branch": "main"}, "deployment": map[string]any{"deploymentId": "d_previous"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := make(map[string]any, len(base)+len(tc.source))
			for key, value := range base {
				body[key] = value
			}
			for key, value := range tc.source {
				body[key] = value
			}

			res := testutil.CallRoute[map[string]any, openapi.BadRequestErrorResponse](h, route, authHeaders(setup.RootKey), body)
			require.Equal(t, http.StatusBadRequest, res.Status, "received: %s", res.RawBody)
		})
	}
}
