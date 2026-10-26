package project_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/project"
)

// TestDeleteMissingProjectDoesNotRetry proves a delete of a project that no
// longer exists finishes. The lookup error must be terminal: restate.Run retries
// its closure on any non-terminal error, so returning the raw sql.ErrNoRows
// leaves the invocation retrying forever.
func TestDeleteMissingProjectDoesNotRetry(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)

	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	svc, err := project.New(project.Config{DB: database, Auditlogs: auditlogSvc})
	require.NoError(t, err)

	restateCfg := containers.Restate(t, hydrav1.NewProjectServiceServer(svc))
	admin := restateadmin.New(restateadmin.Config{BaseURL: restateCfg.AdminURL, APIKey: ""})

	sent, err := hydrav1.NewProjectServiceIngressClient(restateCfg.IngressClient, uid.New(uid.ProjectPrefix)).
		Delete().
		Send(ctx, &hydrav1.DeleteProjectRequest{
			Actor:         &ctrlv1.ActorInfo{Id: uid.New("user"), Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
			CorrelationId: uid.New("corr"),
		})
	require.NoError(t, err)

	requireTerminal(t, ctx, admin, sent.Id())
}

// TestDeleteTwiceIsIdempotent proves a second delete of the same project
// finishes. The first delete removes the row, so the second one finds nothing
// and must treat that as success.
func TestDeleteTwiceIsIdempotent(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	t.Cleanup(cancel)

	database, err := db.New(containers.MySQL(t).DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	created := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: seeder.Resources.UserWorkspace.ID,
		Name:        "KEBAP",
		Slug:        "kebap-" + uid.New(uid.TestPrefix),
	})

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)
	svc, err := project.New(project.Config{DB: database, Auditlogs: auditlogSvc})
	require.NoError(t, err)

	restateCfg := containers.Restate(t, hydrav1.NewProjectServiceServer(svc))
	admin := restateadmin.New(restateadmin.Config{BaseURL: restateCfg.AdminURL, APIKey: ""})

	req := &hydrav1.DeleteProjectRequest{
		Actor:         &ctrlv1.ActorInfo{Id: uid.New("user"), Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
		CorrelationId: uid.New("corr"),
	}

	// Both go out before either finishes, which is the double-click the API
	// allows: it returns 202 as soon as the delete is enqueued, so a second
	// click arrives while the first is still running.
	first, err := hydrav1.NewProjectServiceIngressClient(restateCfg.IngressClient, created.ID).Delete().Send(ctx, req)
	require.NoError(t, err)
	second, err := hydrav1.NewProjectServiceIngressClient(restateCfg.IngressClient, created.ID).Delete().Send(ctx, req)
	require.NoError(t, err)

	requireTerminal(t, ctx, admin, first.Id())
	requireTerminal(t, ctx, admin, second.Id())

	_, err = database.FindProjectById(ctx, created.ID)
	require.True(t, db.IsNotFound(err), "the project row must be gone")
}

// requireTerminal fails unless the invocation stops being live within the
// window. FindLiveInvocations reports every invocation that has not reached
// status completed, so a retrying one stays live.
func requireTerminal(t *testing.T, ctx context.Context, admin *restateadmin.Client, invocationID string) {
	t.Helper()
	require.NotEmpty(t, invocationID)

	require.Eventually(t, func() bool {
		live, err := admin.FindLiveInvocations(ctx, []string{invocationID})
		require.NoError(t, err)
		return !live[invocationID]
	}, 20*time.Second, 250*time.Millisecond,
		"invocation %s is still retrying; the handler must treat a missing row as terminal", invocationID)
}
