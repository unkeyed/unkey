package deployment

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TestToResponseDetailedGating guards the list/get split: a failed deployment in
// a list must not report a bogus `unknown` failure just because its steps were
// never loaded.
func TestToResponseDetailedGating(t *testing.T) {
	failedDep := db.Deployment{
		ID:            "d_KEBAP",
		Status:        db.DeploymentsStatusFailed,
		DesiredState:  db.DeploymentsDesiredStateRunning,
		EnvironmentID: "env_1",
		AppID:         "app_1",
		ProjectID:     "proj_1",
	}

	t.Run("list item omits error and domains", func(t *testing.T) {
		got := ToResponse(Input{Deployment: failedDep, Detailed: false})
		require.Nil(t, got.Error)
		require.Nil(t, got.Domains)
	})

	t.Run("detailed failed deployment includes error and empty domains", func(t *testing.T) {
		got := ToResponse(Input{
			Deployment: failedDep,
			Detailed:   true,
			Steps: []db.DeploymentStep{
				{Step: db.DeploymentStepsStepDeploying, Error: sql.NullString{Valid: true, String: deployfail.MsgNoSchedulableRegions}},
			},
			Domains: nil,
		})
		require.NotNil(t, got.Error)
		require.Equal(t, openapi.DeploymentErrorCodeNoSchedulableRegions, got.Error.Code)
		require.NotNil(t, got.Domains)
		require.Empty(t, *got.Domains)
	})
}

// TestToResponseRegions guards the required regions field: it must marshal as a
// present slice (never nil) and pass through the configured region names.
func TestToResponseRegions(t *testing.T) {
	dep := db.Deployment{ID: "d_KEBAP", Status: db.DeploymentsStatusReady}

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
	t.Run("git-sourced sets git, not docker", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:           "d_1",
			GitCommitSha: sql.NullString{Valid: true, String: "9f2c1a7d3b"},
			GitBranch:    sql.NullString{Valid: true, String: "main"},
			// git builds also fill image with the built output; must not leak as docker
			Image: sql.NullString{Valid: true, String: "ghcr.io/built/output:sha"},
		}})
		require.NotNil(t, got.Git)
		require.Equal(t, "9f2c1a7d3b", got.Git.CommitSha)
		require.NotNil(t, got.Git.Branch)
		require.Equal(t, "main", *got.Git.Branch)
		require.Nil(t, got.Docker)
	})

	t.Run("image-sourced sets docker, not git", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:    "d_2",
			Image: sql.NullString{Valid: true, String: "ghcr.io/acme/api:v1.2.3"},
		}})
		require.NotNil(t, got.Docker)
		require.Equal(t, "ghcr.io/acme/api:v1.2.3", got.Docker.Image)
		require.Nil(t, got.Git)
	})

	t.Run("git without branch omits branch", func(t *testing.T) {
		got := ToResponse(Input{Deployment: db.Deployment{
			ID:           "d_3",
			GitCommitSha: sql.NullString{Valid: true, String: "abc"},
		}})
		require.NotNil(t, got.Git)
		require.Nil(t, got.Git.Branch)
	})
}

func TestToResponseIsCurrent(t *testing.T) {
	dep := db.Deployment{
		ID:           "d_1",
		Status:       db.DeploymentsStatusReady,
		DesiredState: db.DeploymentsDesiredStateRunning,
	}

	t.Run("app points here", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, AppCurrentDeploymentID: "d_1", AppIsRolledBack: false})
		require.True(t, got.IsCurrent)
	})
	t.Run("app points here even when rolled back (still serves traffic)", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, AppCurrentDeploymentID: "d_1", AppIsRolledBack: true})
		require.True(t, got.IsCurrent)
	})
	t.Run("app points elsewhere", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, AppCurrentDeploymentID: "d_other"})
		require.False(t, got.IsCurrent)
	})
	t.Run("app has no current deployment", func(t *testing.T) {
		got := ToResponse(Input{Deployment: dep, AppCurrentDeploymentID: ""})
		require.False(t, got.IsCurrent)
	})
}
