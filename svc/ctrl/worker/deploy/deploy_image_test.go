package deploy_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/internal/services/flags"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/worker/deploy"
)

// TestDeployRejectsPrivateRegistryDigest prevents immutable references from
// bypassing the registry transport's destination policy.
func TestDeployRejectsPrivateRegistryDigest(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	t.Cleanup(cancel)
	database, fixture := newDeployFixture(t, ctx)
	row := fixture.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:          uid.New(uid.DeploymentPrefix),
		WorkspaceID: fixture.workspaceID, ProjectID: fixture.projectID,
		AppID: fixture.appID, EnvironmentID: fixture.environmentID,
		Status: mysqltype.DeploymentsStatusPending,
	})
	audit, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	resolver, err := deploy.NewImageResolver(deploy.RegistryConfig{})
	require.NoError(t, err)
	workflow, err := deploy.New(deploy.Config{
		DB: database, Auditlogs: audit, Flags: flags.NewNoop(), GitHub: githubclient.NewNoop(),
		ImageResolver: resolver,
		DashboardURL:  "https://app.unkey.com", DefaultDomain: "unkey.app",
	})
	require.NoError(t, err)
	cfg := containers.Restate(t, hydrav1.NewDeployWorkflowServer(workflow))
	invocation, err := hydrav1.NewDeployWorkflowIngressClient(cfg.IngressClient, row.ID).Submit(ctx, &hydrav1.DeployRequest{
		DeploymentId: row.ID,
		Source: &hydrav1.DeployRequest_OciImage{OciImage: &hydrav1.OciImage{
			Image: "10.0.0.5:443/acme/api@sha256:" + strings.Repeat("ab", 32),
		}},
	})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		after, findErr := database.FindDeploymentById(ctx, row.ID)
		return findErr == nil && (after.Status == mysqltype.DeploymentsStatusFailed || after.ImageResolved.Valid)
	}, 20*time.Second, 50*time.Millisecond)

	after, err := database.FindDeploymentById(ctx, row.ID)
	require.NoError(t, err)
	require.False(t, after.ImageResolved.Valid, "a blocked image must never become deployable")
	require.Equal(t, mysqltype.DeploymentsStatusFailed, after.Status)
	_, err = invocation.Attach(ctx)
	require.Error(t, err)
}
