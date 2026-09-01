package deploycancel

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/mysql/sqlcomment"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/integration/seed"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// TestCancelAbortsTargets pins the whole contract of one Cancel call: active
// steps carry the reason, active rows transition while terminal rows are left
// alone, only targets with an invocation id reach the canceler, and one audit
// entry lands per target with the caller's actor.
func TestCancelAbortsTargets(t *testing.T) {
	ctx := context.Background()
	f := newCancelFixture(t, ctx)

	building := f.deployment(ctx, mysqltype.DeploymentsStatusBuilding)
	f.startStep(t, ctx, building)
	pending := f.deployment(ctx, mysqltype.DeploymentsStatusPending)
	finished := f.deployment(ctx, mysqltype.DeploymentsStatusReady)

	buildingInvocation := uid.New("inv")
	canceler := &recordingCanceler{}
	actorID := uid.New("user")

	err := Cancel(ctx, f.database, canceler, Params{
		Targets: []Target{
			{ID: building.ID, InvocationID: buildingInvocation},
			{ID: pending.ID, InvocationID: ""},
			{ID: finished.ID, InvocationID: uid.New("inv")},
		},
		Reason: "KEBAP",
		Status: mysqltype.DeploymentsStatusCancelled,
		Audit: &Audit{
			Service:       f.auditlogs,
			Actor:         &ctrlv1.ActorInfo{Id: actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
			CorrelationID: uid.New("corr"),
			WorkspaceID:   f.workspaceID,
			Meta:          map[string]any{"appId": f.appID},
		},
	})
	require.NoError(t, err)

	var stepError sql.NullString
	require.NoError(t, f.database.RO().QueryRowContext(ctx,
		"SELECT error FROM deployment_steps WHERE deployment_id = ?", building.ID,
	).Scan(&stepError))
	require.Equal(t, "KEBAP", stepError.String, "the in-flight step must carry the cancel reason")

	f.requireStatus(t, ctx, building.ID, mysqltype.DeploymentsStatusCancelled)
	f.requireStatus(t, ctx, pending.ID, mysqltype.DeploymentsStatusCancelled)
	f.requireStatus(t, ctx, finished.ID, mysqltype.DeploymentsStatusReady,
		"a terminal row must never be rewritten by a late cancel")

	// The ready row's invocation is still cancelled: its terminal STATUS is
	// what the guard protects, while the invocation may be a compensation
	// still unwinding, and killing it is the caller's explicit intent.
	require.Len(t, canceler.cancelled, 2)
	require.Contains(t, canceler.cancelled, buildingInvocation)

	require.Equal(t, 1, f.countAudits(t, ctx, building.ID, actorID))
	require.Equal(t, 1, f.countAudits(t, ctx, pending.ID, actorID))
}

// TestCancelReturnsInvocationErrorsBeforeAuditing pins the retry story: a
// failed invocation cancel surfaces as an error and suppresses the audit
// entries, so the caller's retry (which re-runs the whole call) writes them
// exactly once, after every invocation is actually dead.
func TestCancelReturnsInvocationErrorsBeforeAuditing(t *testing.T) {
	ctx := context.Background()
	f := newCancelFixture(t, ctx)

	deployment := f.deployment(ctx, mysqltype.DeploymentsStatusBuilding)
	actorID := uid.New("user")
	canceler := &recordingCanceler{fail: true}

	params := Params{
		Targets: []Target{{ID: deployment.ID, InvocationID: uid.New("inv")}},
		Reason:  "KEBAP",
		Status:  mysqltype.DeploymentsStatusCancelled,
		Audit: &Audit{
			Service:     f.auditlogs,
			Actor:       &ctrlv1.ActorInfo{Id: actorID, Type: ctrlv1.ActorType_ACTOR_TYPE_USER},
			WorkspaceID: f.workspaceID,
		},
	}
	require.Error(t, Cancel(ctx, f.database, canceler, params))

	f.requireStatus(t, ctx, deployment.ID, mysqltype.DeploymentsStatusCancelled,
		"the status flip must land even when the invocation cancel fails")
	require.Equal(t, 0, f.countAudits(t, ctx, deployment.ID, actorID),
		"a failed pass must not audit work it did not finish")

	canceler.fail = false
	require.NoError(t, Cancel(ctx, f.database, canceler, params))
	require.Equal(t, 1, f.countAudits(t, ctx, deployment.ID, actorID))
}

// TestCancelWithoutActorWritesNoAudit: sibling dedup passes no audit spec, and
// a caller whose request carried no actor must not fabricate a system entry.
func TestCancelWithoutActorWritesNoAudit(t *testing.T) {
	ctx := context.Background()
	f := newCancelFixture(t, ctx)

	deployment := f.deployment(ctx, mysqltype.DeploymentsStatusPending)

	require.NoError(t, Cancel(ctx, f.database, &recordingCanceler{}, Params{
		Targets: []Target{{ID: deployment.ID, InvocationID: ""}},
		Reason:  "KEBAP",
		Status:  mysqltype.DeploymentsStatusSuperseded,
		Audit:   &Audit{Service: f.auditlogs, Actor: nil, WorkspaceID: f.workspaceID},
	}))

	f.requireStatus(t, ctx, deployment.ID, mysqltype.DeploymentsStatusSuperseded)
	rows, err := f.database.ListClickhouseOutboxByWorkspace(ctx, f.workspaceID)
	require.NoError(t, err)
	require.Empty(t, rows)
}

// recordingCanceler stands in for the Restate admin API.
type recordingCanceler struct {
	cancelled []string
	fail      bool
}

func (c *recordingCanceler) CancelInvocation(_ context.Context, invocationID string) error {
	if c.fail {
		return errors.New("KEBAP: admin unavailable")
	}
	c.cancelled = append(c.cancelled, invocationID)
	return nil
}

type cancelFixture struct {
	database  db.Database
	seeder    *seed.Seeder
	auditlogs auditlogs.AuditLogService

	workspaceID   string
	projectID     string
	appID         string
	environmentID string
}

func newCancelFixture(t *testing.T, ctx context.Context) *cancelFixture {
	t.Helper()

	mysqlCfg := containers.MySQL(t)
	database, err := db.New(mysqlCfg.DSN, sqlcomment.Disabled())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })

	auditlogSvc, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	seeder := seed.New(t, database, nil)
	seeder.Seed(ctx)
	workspaceID := seeder.Resources.UserWorkspace.ID

	project := seeder.CreateProject(ctx, seed.CreateProjectRequest{
		ID:          uid.New(uid.ProjectPrefix),
		WorkspaceID: workspaceID,
		Name:        "KEBAP",
		Slug:        strings.ToLower(strings.ReplaceAll(uid.New(uid.ProjectPrefix), "_", "-")),
	})
	app := seeder.CreateApp(ctx, seed.CreateAppRequest{
		ID:            uid.New(uid.AppPrefix),
		WorkspaceID:   workspaceID,
		ProjectID:     project.ID,
		Name:          "KEBAP",
		Slug:          strings.ToLower(strings.ReplaceAll(uid.New(uid.AppPrefix), "_", "-")),
		DefaultBranch: "main",
	})
	environment := seeder.CreateEnvironment(ctx, seed.CreateEnvironmentRequest{
		ID:          uid.New(uid.EnvironmentPrefix),
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		AppID:       app.ID,
		Slug:        "preview",
		Kind:        mysqltype.EnvironmentKindPreview,
	})

	return &cancelFixture{
		database:      database,
		seeder:        seeder,
		auditlogs:     auditlogSvc,
		workspaceID:   workspaceID,
		projectID:     project.ID,
		appID:         app.ID,
		environmentID: environment.ID,
	}
}

func (f *cancelFixture) deployment(ctx context.Context, status mysqltype.DeploymentsStatus) db.Deployment {
	return f.seeder.CreateDeployment(ctx, seed.CreateDeploymentRequest{
		ID:            uid.New(uid.DeploymentPrefix),
		WorkspaceID:   f.workspaceID,
		ProjectID:     f.projectID,
		AppID:         f.appID,
		EnvironmentID: f.environmentID,
		Status:        status,
	})
}

func (f *cancelFixture) startStep(t *testing.T, ctx context.Context, deployment db.Deployment) {
	t.Helper()
	require.NoError(t, f.database.InsertDeploymentStep(ctx, db.InsertDeploymentStepParams{
		WorkspaceID:   deployment.WorkspaceID,
		ProjectID:     deployment.ProjectID,
		AppID:         deployment.AppID,
		EnvironmentID: deployment.EnvironmentID,
		DeploymentID:  deployment.ID,
		Step:          db.DeploymentStepsStepBuilding,
		StartedAt:     uint64(deployment.CreatedAt),
	}))
}

func (f *cancelFixture) requireStatus(
	t *testing.T,
	ctx context.Context,
	deploymentID string,
	want mysqltype.DeploymentsStatus,
	msgAndArgs ...any,
) {
	t.Helper()
	row, err := f.database.FindDeploymentById(ctx, deploymentID)
	require.NoError(t, err)
	require.Equal(t, want, row.Status, msgAndArgs...)
}

func (f *cancelFixture) countAudits(t *testing.T, ctx context.Context, deploymentID, actorID string) int {
	t.Helper()
	rows, err := f.database.ListClickhouseOutboxByWorkspace(ctx, f.workspaceID)
	require.NoError(t, err)

	count := 0
	for _, row := range rows {
		payload := string(row.Payload)
		if strings.Contains(payload, string(auditlog.DeploymentCancelEvent)) &&
			strings.Contains(payload, deploymentID) &&
			strings.Contains(payload, actorID) {
			count++
		}
	}
	return count
}
