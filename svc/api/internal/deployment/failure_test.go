package deployment

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// TestClassifyFailure locks the two classification paths: a build-step failure
// is build_failed by step (its message is not stable across the Restate
// boundary), and every other step is matched against the shared deployfail
// constants the worker writes.
func TestClassifyFailure(t *testing.T) {
	const deploying = db.DeploymentStepsStepDeploying
	const starting = db.DeploymentStepsStepStarting

	cases := []struct {
		name    string
		step    db.DeploymentStepsStep
		message string
		want    openapi.DeploymentFailureCode
	}{
		{"build step ignores message", db.DeploymentStepsStepBuilding, "any opaque depot error", openapi.DeploymentFailureCodeBuildFailed},
		{"build step even with empty message", db.DeploymentStepsStepBuilding, "", openapi.DeploymentFailureCodeBuildFailed},
		{"regions", deploying, deployfail.MsgNoSchedulableRegions, openapi.DeploymentFailureCodeNoSchedulableRegions},
		{"cpu quota", deploying, deployfail.MsgCPUQuotaExceeded, openapi.DeploymentFailureCodeCpuQuotaExceeded},
		{"memory quota", deploying, deployfail.MsgMemoryQuotaExceeded, openapi.DeploymentFailureCodeMemoryQuotaExceeded},
		{"storage quota", deploying, deployfail.MsgStorageQuotaExceeded, openapi.DeploymentFailureCodeStorageQuotaExceeded},
		{"port too low", starting, deployfail.MsgPortTooLow, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
		{"port too high", starting, deployfail.MsgPortTooHigh, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
		{"cpu too low", starting, deployfail.MsgCPUTooLow, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
		{"memory too low", starting, deployfail.MsgMemoryTooLow, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
		// UserFacingMessage joins public messages, so a match must survive being
		// embedded in a longer string.
		{"joined with wrapper", deploying, "Regional deployment targets could not be prepared. " + deployfail.MsgNoSchedulableRegions, openapi.DeploymentFailureCodeNoSchedulableRegions},
		{"unclassified non-build", deploying, "Instances did not become healthy in time.", openapi.DeploymentFailureCodeUnknown},
		{"empty non-build", deploying, "", openapi.DeploymentFailureCodeUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, classifyFailure(tc.step, tc.message))
		})
	}
}

func TestDeriveFailure(t *testing.T) {
	t.Run("non-failed status returns nil", func(t *testing.T) {
		require.Nil(t, deriveFailure(db.DeploymentsStatusReady, nil))
		require.Nil(t, deriveFailure(db.DeploymentsStatusBuilding, nil))
	})

	t.Run("failed with no recorded error falls back to unknown", func(t *testing.T) {
		f := deriveFailure(db.DeploymentsStatusFailed, nil)
		require.NotNil(t, f)
		require.Equal(t, openapi.DeploymentFailureCodeUnknown, f.Code)
		require.NotEmpty(t, f.Message)
	})

	t.Run("failed build step is build_failed regardless of message", func(t *testing.T) {
		steps := []db.DeploymentStep{
			{Step: db.DeploymentStepsStepBuilding, Error: sql.NullString{Valid: true, String: "opaque depot build output"}},
		}
		f := deriveFailure(db.DeploymentsStatusFailed, steps)
		require.NotNil(t, f)
		require.Equal(t, openapi.DeploymentFailureCodeBuildFailed, f.Code)
		require.Equal(t, "building", f.Step)
	})

	t.Run("failed deploying step is classified by message", func(t *testing.T) {
		steps := []db.DeploymentStep{
			{Step: db.DeploymentStepsStepDeploying, Error: sql.NullString{Valid: true, String: deployfail.MsgNoSchedulableRegions}},
		}
		f := deriveFailure(db.DeploymentsStatusFailed, steps)
		require.NotNil(t, f)
		require.Equal(t, openapi.DeploymentFailureCodeNoSchedulableRegions, f.Code)
		require.Equal(t, "deploying", f.Step)
	})
}
