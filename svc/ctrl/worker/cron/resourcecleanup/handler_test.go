package resourcecleanup

import (
	"strings"
	"testing"

	restate "github.com/restatedev/sdk-go"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestCleanupContinuesAfterFailureAndResumesBoundedScan(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	projectID, appID, envID, deploymentID := "project", "app", "environment", "deployment"
	f.project(projectID)
	f.app(appID, projectID)
	f.environment(envID, appID)
	f.exec("UPDATE environments SET project_id = ? WHERE id = ?", projectID, envID)
	f.deployment(deploymentID, appID)
	f.exec("UPDATE deployments SET environment_id = ? WHERE id = ?", envID, deploymentID)
	f.app("orphan-app", "missing-project")
	f.exec("CREATE TRIGGER reject_app_cleanup BEFORE DELETE ON apps FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'injected deletion failure'")

	values := make([]string, 0, batchSize*batchesPerRun+1)
	args := make([]any, 0, 3*(batchSize*batchesPerRun+1))
	for i := range batchSize*batchesPerRun + 1 {
		owner := deploymentID
		if i == batchSize*batchesPerRun {
			owner = "missing-deployment"
		}
		id := uid.New(uid.FrontlineRoutePrefix)
		values = append(values, "(?, '', '', ?, '', ?, 1)")
		args = append(args, id, owner, id)
	}
	f.exec("INSERT INTO frontline_routes (id, project_id, app_id, deployment_id, environment_id, fully_qualified_domain_name, created_at) VALUES "+strings.Join(values, ","), args...)
	f.exec("INSERT INTO app_source_oci (workspace_id, app_id, image_reference, created_at) VALUES (?, 'missing-app', 'nginx', 1)", f.workspaceID)

	h, err := New(Config{DB: f.db, Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)
	runtime := containers.Restate(t, restate.NewObject("hydra.v1.CronService").Handler("RunResourceCleanup", restate.NewObjectHandler(h.Handle, RetryPolicy())))
	client := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, objectKey).RunResourceCleanup()
	_, err = client.Request(ctx, &hydrav1.RunResourceCleanupRequest{})
	require.Error(t, err)
	require.Empty(t, f.pks("app_source_oci"), "an app failure must not prevent independent tables from being cleaned")
	require.Len(t, f.pks("frontline_routes"), batchSize*batchesPerRun+1, "one invocation must stop at its batch limit")

	f.exec("DROP TRIGGER reject_app_cleanup")
	result, err := client.Request(ctx, &hydrav1.RunResourceCleanupRequest{})
	require.NoError(t, err)
	require.EqualValues(t, 2, result.GetRowsDeleted(), "the next invocation must resume beyond the healthy prefix and remove the orphan app and route")
	require.Len(t, f.pks("frontline_routes"), batchSize*batchesPerRun)
}
