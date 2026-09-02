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
	failedDep := db.Deployment{
		ID:            uid.New(uid.DeploymentPrefix),
		Status:        mysqltype.DeploymentsStatusFailed,
		DesiredState:  mysqltype.DeploymentsDesiredStateRunning,
		EnvironmentID: uid.New(uid.EnvironmentPrefix),
		AppID:         uid.New(uid.AppPrefix),
		ProjectID:     uid.New(uid.ProjectPrefix),
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
	dep := db.Deployment{ID: uid.New(uid.DeploymentPrefix), Status: mysqltype.DeploymentsStatusReady}

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
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:           uid.New(uid.DeploymentPrefix),
			Source:       db.DeploymentsSourceGit,
			GitCommitSha: sql.NullString{Valid: true, String: "9f2c1a7d3b"},
			GitBranch:    sql.NullString{Valid: true, String: "main"},
			Image:        sql.NullString{Valid: true, String: "ghcr.io/built/output:sha"},
		}})
		require.NotNil(t, got.Git)
		require.Equal(t, "9f2c1a7d3b", got.Git.CommitSha)
		require.NotNil(t, got.Git.Branch)
		require.Equal(t, "main", *got.Git.Branch)
		require.Nil(t, got.Docker)
	})

	t.Run("OCI-sourced sets compatibility field, not git", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:             uid.New(uid.DeploymentPrefix),
			Source:         db.DeploymentsSourceOci,
			ImageRequested: sql.NullString{Valid: true, String: "ghcr.io/acme/api:v1.2.3"},
			Image:          sql.NullString{Valid: true, String: "ghcr.io/acme/api@sha256:resolved"},
		}})
		require.NotNil(t, got.Docker)
		require.Equal(t, "ghcr.io/acme/api:v1.2.3", got.Docker.Image)
		require.Nil(t, got.Git)
	})

	t.Run("resolved image prefers additive column", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:            uid.New(uid.DeploymentPrefix),
			Source:        db.DeploymentsSourceOci,
			Image:         sql.NullString{Valid: true, String: "ghcr.io/acme/api@sha256:legacy"},
			ImageResolved: sql.NullString{Valid: true, String: "ghcr.io/acme/api@sha256:resolved"},
		}})
		require.NotNil(t, got.Docker)
		require.Equal(t, "ghcr.io/acme/api@sha256:resolved", got.Docker.Image)
	})

	t.Run("invalid requested image falls back to resolved image", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:             uid.New(uid.DeploymentPrefix),
			Source:         db.DeploymentsSourceOci,
			ImageRequested: sql.NullString{Valid: false, String: "ghcr.io/acme/api:invalid"},
			ImageResolved:  sql.NullString{Valid: true, String: "ghcr.io/acme/api@sha256:resolved"},
		}})
		require.NotNil(t, got.Docker)
		require.Equal(t, "ghcr.io/acme/api@sha256:resolved", got.Docker.Image)
	})

	t.Run("git without branch omits branch", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:           uid.New(uid.DeploymentPrefix),
			Source:       db.DeploymentsSourceGit,
			GitCommitSha: sql.NullString{Valid: true, String: "abc"},
		}})
		require.NotNil(t, got.Git)
		require.Nil(t, got.Git.Branch)
	})

	t.Run("unknown source remains neutral", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:             uid.New(uid.DeploymentPrefix),
			GitCommitSha:   sql.NullString{Valid: true, String: "abc"},
			Source:         db.DeploymentsSourceUnknown,
			ImageRequested: sql.NullString{Valid: true, String: "nginx:stable"},
			Image:          sql.NullString{Valid: true, String: "nginx@sha256:resolved"},
		}})
		require.Nil(t, got.Git)
		require.Nil(t, got.Docker)
	})

	t.Run("unsupported source remains neutral", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:             uid.New(uid.DeploymentPrefix),
			Source:         db.DeploymentsSource("future_source"),
			GitCommitSha:   sql.NullString{Valid: true, String: "abc"},
			ImageRequested: sql.NullString{Valid: true, String: "nginx:stable"},
			Image:          sql.NullString{Valid: true, String: "nginx@sha256:resolved"},
		}})
		require.Nil(t, got.Git)
		require.Nil(t, got.Docker)
	})
}

func TestToResponseIsCurrent(t *testing.T) {
	dep := db.Deployment{
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
