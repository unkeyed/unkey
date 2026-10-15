package resourcecleanup

import (
	"strings"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/restatedev/sdk-go/ingress"
	"github.com/stretchr/testify/require"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestEnvironmentRepairGracePeriod(t *testing.T) {
	f := newFixture(t)
	for _, parent := range []string{"environment", "app", "project"} {
		for _, age := range []time.Duration{0, 119 * time.Minute, 121 * time.Minute} {
			t.Run(parent+"_"+age.String(), func(t *testing.T) {
				projectID, appID, envID := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix), uid.New(uid.EnvironmentPrefix)
				f.project(projectID)
				f.app(appID, projectID)
				f.environment(envID, appID)
				f.exec("UPDATE environments SET project_id = ? WHERE id = ?", projectID, envID)
				id := map[string]string{"environment": envID, "app": appID, "project": projectID}[parent]
				if age != 0 {
					f.exec("UPDATE "+parent+"s SET deleted_at = ? WHERE id = ?", time.Now().Add(-age).UnixMilli(), id)
				}
				rows, err := f.db.ScanEnvironmentsForCleanup(t.Context(), db.ScanEnvironmentsForCleanupParams{Limit: batchSize})
				require.NoError(t, err)
				found := false
				for _, row := range rows {
					if row.ID == envID {
						found = true
						require.Equal(t, age > 2*time.Hour, row.Orphaned != 0)
					}
				}
				require.True(t, found)
				if parent != "environment" {
					f.exec("DELETE FROM "+parent+"s WHERE id = ?", id)
					rows, err = f.db.ScanEnvironmentsForCleanup(t.Context(), db.ScanEnvironmentsForCleanupParams{Limit: batchSize})
					require.NoError(t, err)
					for _, row := range rows {
						if row.ID == envID {
							require.NotZero(t, row.Orphaned, "missing parents must bypass the grace period")
						}
					}
				}
			})
		}
	}
}

func TestCleanupPreservesRecentParentsAndReapsAgedEmptyParents(t *testing.T) {
	f := newFixture(t)
	kept := map[string][]uint64{}
	for _, target := range []string{"apps", "projects"} {
		for _, age := range []time.Duration{119 * time.Minute, 121 * time.Minute} {
			projectID, appID := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix)
			projectPK := f.project(projectID)
			appPK := f.app(appID, projectID)
			id := appID
			if target == "projects" {
				id = projectID
				emptyProjectID := uid.New(uid.ProjectPrefix)
				emptyProjectPK := f.project(emptyProjectID)
				f.exec("UPDATE projects SET deleted_at = ? WHERE id = ?", time.Now().Add(-age).UnixMilli(), emptyProjectID)
				if age < 2*time.Hour {
					kept["projects"] = append(kept["projects"], emptyProjectPK)
				}
			}
			f.exec("UPDATE "+target+" SET deleted_at = ? WHERE id = ?", time.Now().Add(-age).UnixMilli(), id)
			if age < 2*time.Hour || target == "apps" {
				kept["projects"] = append(kept["projects"], projectPK)
			}
			if age < 2*time.Hour {
				kept["apps"] = append(kept["apps"], appPK)
			}
		}
	}
	for _, child := range []string{"environment", "deployment"} {
		projectID, appID := uid.New(uid.ProjectPrefix), uid.New(uid.AppPrefix)
		kept["projects"] = append(kept["projects"], f.project(projectID))
		kept["apps"] = append(kept["apps"], f.app(appID, projectID))
		f.exec("UPDATE projects SET deleted_at = 1 WHERE id = ?", projectID)
		f.exec("UPDATE apps SET deleted_at = 1 WHERE id = ?", appID)
		if child == "environment" {
			f.environment(uid.New(uid.EnvironmentPrefix), appID)
		} else {
			f.deployment(uid.New(uid.DeploymentPrefix), appID)
		}
	}
	for _, child := range []string{"environments", "deployments"} {
		projectID := uid.New(uid.ProjectPrefix)
		kept["projects"] = append(kept["projects"], f.project(projectID))
		f.exec("UPDATE projects SET deleted_at = 1 WHERE id = ?", projectID)
		if child == "environments" {
			f.environment(uid.New(uid.EnvironmentPrefix), "missing-app")
		} else {
			f.deployment(uid.New(uid.DeploymentPrefix), "missing-app")
		}
		f.exec("UPDATE "+child+" SET project_id = ? WHERE app_id = 'missing-app'", projectID)
	}
	h, err := New(Config{DB: f.db, Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)
	runtime := containers.Restate(t,
		restate.NewObject("hydra.v1.CronService").Handler("RunResourceCleanup", restate.NewObjectHandler(h.Handle, RetryPolicy())),
		restate.NewObject("hydra.v1.EnvironmentService").Handler("Delete", restate.NewObjectHandler(func(restate.ObjectContext, *hydrav1.DeleteEnvironmentRequest) (*hydrav1.DeleteEnvironmentResponse, error) {
			return nil, restate.TerminalErrorf("deletion workflow unavailable")
		})),
	)
	_, err = hydrav1.NewCronServiceIngressClient(runtime.IngressClient, objectKey).RunResourceCleanup().Request(t.Context(), &hydrav1.RunResourceCleanupRequest{})
	require.NoError(t, err)
	require.ElementsMatch(t, kept["apps"], f.pks("apps"))
	require.ElementsMatch(t, kept["projects"], f.pks("projects"))
}

func TestCleanupDeduplicatesFailedEnvironmentRepairs(t *testing.T) {
	f := newFixture(t)
	envID := uid.New(uid.EnvironmentPrefix)
	f.environment(envID, "missing-app")
	h, err := New(Config{DB: f.db, Heartbeat: healthcheck.NewNoop()})
	require.NoError(t, err)
	runtime := containers.Restate(t,
		restate.NewObject("hydra.v1.CronService").Handler("RunResourceCleanup", restate.NewObjectHandler(h.Handle, RetryPolicy())),
		restate.NewObject("hydra.v1.EnvironmentService").Handler("Delete", restate.NewObjectHandler(func(ctx restate.ObjectContext, _ *hydrav1.DeleteEnvironmentRequest) (*hydrav1.DeleteEnvironmentResponse, error) {
			attempts, err := restate.Get[int](ctx, "attempts")
			if err != nil {
				return nil, err
			}
			restate.Set(ctx, "attempts", attempts+1)
			return nil, restate.TerminalErrorf("injected repair failure")
		})).Handler("Attempts", restate.NewObjectHandler(func(ctx restate.ObjectContext, _ restate.Void) (int, error) {
			return restate.Get[int](ctx, "attempts")
		})),
	)
	client := hydrav1.NewCronServiceIngressClient(runtime.IngressClient, objectKey).RunResourceCleanup()
	for range 3 {
		_, err = client.Request(t.Context(), &hydrav1.RunResourceCleanupRequest{})
		require.NoError(t, err)
		attempts, err := ingress.Object[restate.Void, int](runtime.IngressClient, "hydra.v1.EnvironmentService", envID, "Attempts").Request(t.Context(), restate.Void{})
		require.NoError(t, err)
		require.Equal(t, 1, attempts, "repair must reuse the invocation until its idempotency retention expires")
	}
	require.Len(t, f.pks("environments"), 1)
}

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
