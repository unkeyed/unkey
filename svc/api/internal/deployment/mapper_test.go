package deployment

import (
	"database/sql"
	"testing"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TestToResponseError covers failure derivation and domains, which listDeployments
// and getDeployment now populate identically: a failed deployment reports the
// error classified from its failing steps, and domains always serialize as a
// present slice.
func TestToResponseError(t *testing.T) {
	failedDep := db.ListDeploymentsRow{
		ID:           uid.New(uid.DeploymentPrefix),
		Status:       mysqltype.DeploymentsStatusFailed,
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	}

	t.Run("failed deployment reports classified error", func(t *testing.T) {
		got := ToResponse(Input{
			Deployment: failedDep,
			Steps: []db.DeploymentStep{
				{Step: db.DeploymentStepsStepDeploying, Error: sql.NullString{Valid: true, String: deployfail.MsgNoSchedulableRegions}},
			},
		})
		require.NotNil(t, got.Error)
		require.Equal(t, openapi.DeploymentErrorCodeNoSchedulableRegions, got.Error.Code)
	})

	t.Run("non-failed deployment has no error", func(t *testing.T) {
		readyDep := failedDep
		readyDep.Status = mysqltype.DeploymentsStatusReady
		got := ToResponse(Input{Deployment: readyDep})
		require.Nil(t, got.Error)
	})

	t.Run("nil domains become an empty slice, not null", func(t *testing.T) {
		got := ToResponse(Input{Deployment: failedDep, Domains: nil})
		require.NotNil(t, got.Domains)
		require.Empty(t, *got.Domains)
	})
}

// TestToResponseRegions guards the required regions field: it must marshal as a
// present slice (never nil) and pass through the configured region names.
func TestToResponseRegions(t *testing.T) {
	dep := db.ListDeploymentsRow{ID: uid.New(uid.DeploymentPrefix), Status: mysqltype.DeploymentsStatusReady}

	t.Run("populated regions pass through", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, Regions: []string{"us-east-1", "eu-west-1"}})
		require.Equal(t, []string{"us-east-1", "eu-west-1"}, got.Regions)
	})

	t.Run("nil regions become an empty slice, not null", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, Regions: nil})
		require.NotNil(t, got.Regions)
		require.Empty(t, got.Regions)
	})
}

func TestToResponseSource(t *testing.T) {
	t.Run("git-sourced sets git, not OCI compatibility field", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:           uid.New(uid.DeploymentPrefix),
			Source:       db.DeploymentsSourceGit,
			GitCommitSha: sql.NullString{Valid: true, String: "9f2c1a7d3b"},
			GitBranch:    sql.NullString{Valid: true, String: "main"},
			ImageResolved: sql.NullString{
				Valid:  true,
				String: "ghcr.io/built/output:sha",
			},
		}})
		require.NotNil(t, got.Git)
		require.Equal(t, "9f2c1a7d3b", got.Git.CommitSha)
		require.NotNil(t, got.Git.Branch)
		require.Equal(t, "main", *got.Git.Branch)
		require.Nil(t, got.Docker)
	})

	t.Run("OCI-sourced sets compatibility field, not git", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:             uid.New(uid.DeploymentPrefix),
			Source:         db.DeploymentsSourceOci,
			ImageRequested: sql.NullString{Valid: true, String: "ghcr.io/acme/api:v1.2.3"},
			ImageResolved:  sql.NullString{Valid: true, String: "ghcr.io/acme/api@sha256:resolved"},
		}})
		require.NotNil(t, got.Docker)
		require.Equal(t, "ghcr.io/acme/api:v1.2.3", got.Docker.Image)
		require.NotNil(t, got.Docker.ResolvedImage)
		require.Equal(t, "ghcr.io/acme/api@sha256:resolved", *got.Docker.ResolvedImage)
		require.Nil(t, got.Git)
	})

	t.Run("unresolved OCI image omits resolvedImage", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:             uid.New(uid.DeploymentPrefix),
			Source:         db.DeploymentsSourceOci,
			ImageRequested: sql.NullString{Valid: true, String: "ghcr.io/acme/api:v1.2.3"},
		}})
		require.NotNil(t, got.Docker)
		require.Nil(t, got.Docker.ResolvedImage)
	})

	t.Run("git metadata passes through", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:                       uid.New(uid.DeploymentPrefix),
			Source:                   db.DeploymentsSourceGit,
			GitCommitSha:             sql.NullString{Valid: true, String: "9f2c1a7d3b"},
			GitCommitMessage:         sql.NullString{Valid: true, String: "KEBAP: retry on 429"},
			GitCommitTimestamp:       sql.NullInt64{Valid: true, Int64: 1704067100000},
			GitCommitAuthorHandle:    sql.NullString{Valid: true, String: "dana"},
			GitCommitAuthorAvatarUrl: sql.NullString{Valid: true, String: "https://avatars.githubusercontent.com/u/1"},
			PrNumber:                 sql.NullInt64{Valid: true, Int64: 412},
			ForkRepositoryFullName:   sql.NullString{Valid: true, String: "dana/payments-api"},
		}})
		require.Equal(t, &openapi.DeploymentGit{
			CommitSha:              "9f2c1a7d3b",
			CommitMessage:          new("KEBAP: retry on 429"),
			CommitTimestamp:        new(int64(1704067100000)),
			AuthorHandle:           new("dana"),
			AuthorAvatarUrl:        new("https://avatars.githubusercontent.com/u/1"),
			PrNumber:               new(412),
			ForkRepositoryFullName: new("dana/payments-api"),
		}, got.Git)
	})

	t.Run("resolved image is used when requested image is absent", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:            uid.New(uid.DeploymentPrefix),
			Source:        db.DeploymentsSourceOci,
			ImageResolved: sql.NullString{Valid: true, String: "ghcr.io/acme/api@sha256:resolved"},
		}})
		require.NotNil(t, got.Docker)
		require.Equal(t, "ghcr.io/acme/api@sha256:resolved", got.Docker.Image)
	})

	t.Run("invalid requested image falls back to resolved image", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:             uid.New(uid.DeploymentPrefix),
			Source:         db.DeploymentsSourceOci,
			ImageRequested: sql.NullString{Valid: false, String: "ghcr.io/acme/api:invalid"},
			ImageResolved:  sql.NullString{Valid: true, String: "ghcr.io/acme/api@sha256:resolved"},
		}})
		require.NotNil(t, got.Docker)
		require.Equal(t, "ghcr.io/acme/api@sha256:resolved", got.Docker.Image)
	})

	t.Run("git without metadata omits optional fields", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:           uid.New(uid.DeploymentPrefix),
			Source:       db.DeploymentsSourceGit,
			GitCommitSha: sql.NullString{Valid: true, String: "abc"},
		}})
		require.Equal(t, &openapi.DeploymentGit{CommitSha: "abc"}, got.Git)
	})

	t.Run("unknown source remains neutral", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:             uid.New(uid.DeploymentPrefix),
			GitCommitSha:   sql.NullString{Valid: true, String: "abc"},
			Source:         db.DeploymentsSourceUnknown,
			ImageRequested: sql.NullString{Valid: true, String: "nginx:stable"},
			ImageResolved:  sql.NullString{Valid: true, String: "nginx@sha256:resolved"},
		}})
		require.Nil(t, got.Git)
		require.Nil(t, got.Docker)
	})

	t.Run("unsupported source remains neutral", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
			ID:             uid.New(uid.DeploymentPrefix),
			Source:         db.DeploymentsSource("future_source"),
			GitCommitSha:   sql.NullString{Valid: true, String: "abc"},
			ImageRequested: sql.NullString{Valid: true, String: "nginx:stable"},
			ImageResolved:  sql.NullString{Valid: true, String: "nginx@sha256:resolved"},
		}})
		require.Nil(t, got.Git)
		require.Nil(t, got.Docker)
	})
}

func TestToResponseIsCurrent(t *testing.T) {
	dep := db.ListDeploymentsRow{
		ID:           uid.New(uid.DeploymentPrefix),
		Status:       mysqltype.DeploymentsStatusReady,
		DesiredState: mysqltype.DeploymentsDesiredStateRunning,
	}

	current := func(id string) db.ListDeploymentEnvAndAppStateRow {
		return db.ListDeploymentEnvAndAppStateRow{AppCurrentDeploymentID: sql.NullString{Valid: id != "", String: id}}
	}

	t.Run("app points here", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, State: current(dep.ID)})
		require.True(t, got.IsCurrent)
	})
	t.Run("app points here even when rolled back (still serves traffic)", func(t *testing.T) {
		state := current(dep.ID)
		state.AppIsRolledBack = true
		got := ToResponse(Input{Deployment: dep, State: state})
		require.True(t, got.IsCurrent)
	})
	t.Run("app points elsewhere", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, State: current(uid.New(uid.DeploymentPrefix))})
		require.False(t, got.IsCurrent)
	})
	t.Run("app has no current deployment", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, State: current("")})
		require.False(t, got.IsCurrent)
	})
}

func TestToResponseTrigger(t *testing.T) {
	rootKeyID := uid.New(uid.KeyPrefix)
	userID := "user_" + uid.New(uid.TestPrefix)

	cases := []struct {
		name        string
		trigger     db.DeploymentsTrigger
		triggeredBy sql.NullString
		want        *openapi.DeploymentTriggerActor
	}{
		{name: "no actor", trigger: db.DeploymentsTriggerApi, triggeredBy: sql.NullString{}, want: nil},
		{name: "empty actor", trigger: db.DeploymentsTriggerApi, triggeredBy: sql.NullString{Valid: true, String: ""}, want: nil},
		{name: "rebuild operator", trigger: db.DeploymentsTriggerUnkey, triggeredBy: sql.NullString{Valid: true, String: "unkey-ops"}, want: &openapi.DeploymentTriggerActor{Type: openapi.DeploymentTriggerActorTypeSystem, Id: "unkey-ops"}},
		{name: "root key", trigger: db.DeploymentsTriggerCli, triggeredBy: sql.NullString{Valid: true, String: rootKeyID}, want: &openapi.DeploymentTriggerActor{Type: openapi.DeploymentTriggerActorTypeRootKey, Id: rootKeyID}},
		{name: "user", trigger: db.DeploymentsTriggerDashboard, triggeredBy: sql.NullString{Valid: true, String: userID}, want: &openapi.DeploymentTriggerActor{Type: openapi.DeploymentTriggerActorTypeUser, Id: userID}},
		{name: "user id prefix wins over the github trigger", trigger: db.DeploymentsTriggerGithub, triggeredBy: sql.NullString{Valid: true, String: userID}, want: &openapi.DeploymentTriggerActor{Type: openapi.DeploymentTriggerActorTypeUser, Id: userID}},
		{name: "github login", trigger: db.DeploymentsTriggerGithub, triggeredBy: sql.NullString{Valid: true, String: "dana"}, want: &openapi.DeploymentTriggerActor{Type: openapi.DeploymentTriggerActorTypeGithub, Id: "dana"}},
		{name: "unrecognized id", trigger: db.DeploymentsTriggerApi, triggeredBy: sql.NullString{Valid: true, String: "dana"}, want: &openapi.DeploymentTriggerActor{Type: openapi.DeploymentTriggerActorTypeUnknown, Id: "dana"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToResponse(Input{Deployment: db.ListDeploymentsRow{
				ID:          uid.New(uid.DeploymentPrefix),
				Trigger:     tc.trigger,
				TriggeredBy: tc.triggeredBy,
			}})
			require.Equal(t, openapi.DeploymentTriggerVia(tc.trigger), got.Trigger.Via)
			require.Equal(t, tc.want, got.Trigger.Actor)
		})
	}
}

func TestToResponseFinishedAt(t *testing.T) {
	ended := func(at int64) db.DeploymentStep {
		return db.DeploymentStep{Step: db.DeploymentStepsStepBuilding, EndedAt: sql.NullInt64{Valid: true, Int64: at}}
	}
	finalized := db.DeploymentStep{Step: db.DeploymentStepsStepFinalizing, EndedAt: sql.NullInt64{Valid: true, Int64: 4000}}
	open := db.DeploymentStep{Step: db.DeploymentStepsStepDeploying, EndedAt: sql.NullInt64{}}

	cases := []struct {
		name   string
		status mysqltype.DeploymentsStatus
		steps  []db.DeploymentStep
		want   *int64
	}{
		{name: "no steps", status: mysqltype.DeploymentsStatusReady, steps: nil, want: nil},
		{name: "all steps ended", status: mysqltype.DeploymentsStatusReady, steps: []db.DeploymentStep{ended(2000), ended(3000), ended(1000)}, want: new(int64(3000))},
		{name: "failed deployment", status: mysqltype.DeploymentsStatusFailed, steps: []db.DeploymentStep{ended(2000)}, want: new(int64(2000))},
		{name: "a step is open", status: mysqltype.DeploymentsStatusReady, steps: []db.DeploymentStep{ended(2000), open}, want: nil},
		{name: "between steps of a running deployment", status: mysqltype.DeploymentsStatusBuilding, steps: []db.DeploymentStep{ended(2000)}, want: nil},
		{name: "woken deployment keeps its finished pipeline", status: mysqltype.DeploymentsStatusDeploying, steps: []db.DeploymentStep{ended(2000), finalized}, want: new(int64(4000))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToResponse(Input{Deployment: db.ListDeploymentsRow{ID: uid.New(uid.DeploymentPrefix), Status: tc.status}, Steps: tc.steps})
			require.Equal(t, tc.want, got.FinishedAt)
		})
	}
}
