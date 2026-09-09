package handler_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	"github.com/unkeyed/unkey/svc/api/openapi"
	handler "github.com/unkeyed/unkey/svc/api/routes/v3_deployments_create_deployment"
)

func TestCreateOCIDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	restateClient, creates := testutil.RecordingDeployRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Oci:         &openapi.DeploymentSourceOCI{Image: "nginx:latest"},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)
	require.NotEmpty(t, res.Body.Data.DeploymentId)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, res.Body.Data.DeploymentId, observed.DeploymentID)
	require.Equal(t, "nginx:latest", observed.Request.GetImage().GetImage())
	require.Nil(t, observed.Request.GetGit())
}

func TestCreateDeploymentWithAppDefault(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	restateClient, creates := testutil.RecordingDeployRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Nil(t, observed.Request.GetSource(), "no source override must leave the oneof unset")
}

func TestCreateGitDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})
	restateClient, creates := testutil.RecordingDeployRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Git:         &openapi.DeploymentSourceGit{Branch: ptr.P("main")},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)

	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, "main", observed.Request.GetGit().GetCommit().GetBranch())
	require.Nil(t, observed.Request.GetImage())
}

func TestRedeploySendsExistingDeployment(t *testing.T) {
	h := testutil.NewHarness(t)
	setup := h.CreateTestDeploymentSetup(testutil.CreateTestDeploymentSetupOptions{
		Permissions: []string{"environment.*.create_deployment"},
	})

	deployment := h.CreateDeployment(seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   setup.Workspace.ID,
		ProjectID:     setup.Project.ID,
		AppID:         setup.App.ID,
		EnvironmentID: setup.Environment.ID,
	})
	resolved := "ghcr.io/acme/api@sha256:" + strings.Repeat("a", 64)
	_, err := h.DB.RW().ExecContext(context.Background(), `
		UPDATE deployments
		SET source = ?, image_requested = ?, image_resolved = ?
		WHERE id = ?
	`, db.DeploymentsSourceOci, "ghcr.io/acme/api:stable", resolved, deployment.ID)
	require.NoError(t, err)

	restateClient, creates := testutil.RecordingDeployRestate(t)
	route := &handler.Handler{DB: h.DB, Restate: restateClient}
	h.Register(route)

	res := testutil.CallRoute[handler.Request, handler.Response](h, route, authHeaders(setup.RootKey), handler.Request{
		Project:     setup.Project.Slug,
		App:         setup.App.Slug,
		Environment: setup.Environment.Slug,
		Deployment:  &openapi.DeploymentSourceDeployment{DeploymentId: deployment.ID},
	})
	require.Equal(t, http.StatusCreated, res.Status, "received: %s", res.RawBody)

	// The worker resolves the image from the source row; the API only names it
	observed := testutil.Receive(t, creates, 10*time.Second)
	require.Equal(t, deployment.ID, observed.Request.GetExistingDeployment().GetDeploymentId())
	require.Nil(t, observed.Request.GetImage())
}

func authHeaders(rootKey string) http.Header {
	return http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + rootKey},
	}
}
