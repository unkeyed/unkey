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
		want    openapi.DeploymentErrorCode
	}{
		{"build step ignores message", db.DeploymentStepsStepBuilding, "any opaque depot error", openapi.DeploymentErrorCodeBuildFailed},
		{"build step even with empty message", db.DeploymentStepsStepBuilding, "", openapi.DeploymentErrorCodeBuildFailed},
		{"regions", deploying, deployfail.MsgNoSchedulableRegions, openapi.DeploymentErrorCodeNoSchedulableRegions},
		{"cpu quota", deploying, deployfail.MsgCPUQuotaExceeded, openapi.DeploymentErrorCodeCpuQuotaExceeded},
		{"memory quota", deploying, deployfail.MsgMemoryQuotaExceeded, openapi.DeploymentErrorCodeMemoryQuotaExceeded},
		{"storage quota", deploying, deployfail.MsgStorageQuotaExceeded, openapi.DeploymentErrorCodeStorageQuotaExceeded},
		{"port too low", starting, deployfail.MsgPortTooLow, openapi.DeploymentErrorCodeInvalidRuntimeSettings},
		{"port too high", starting, deployfail.MsgPortTooHigh, openapi.DeploymentErrorCodeInvalidRuntimeSettings},
		{"cpu too low", starting, deployfail.MsgCPUTooLow, openapi.DeploymentErrorCodeInvalidRuntimeSettings},
		{"memory too low", starting, deployfail.MsgMemoryTooLow, openapi.DeploymentErrorCodeInvalidRuntimeSettings},
		// UserFacingMessage joins public messages, so a match must survive being
		// embedded in a longer string.
		{"joined with wrapper", deploying, "Regional deployment targets could not be prepared. " + deployfail.MsgNoSchedulableRegions, openapi.DeploymentErrorCodeNoSchedulableRegions},
		{"unclassified non-build", deploying, "Instances did not become healthy in time.", openapi.DeploymentErrorCodeUnknown},
		{"empty non-build", deploying, "", openapi.DeploymentErrorCodeUnknown},
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
		require.Equal(t, openapi.DeploymentErrorCodeUnknown, f.Code)
		require.NotEmpty(t, f.Message)
	})

	t.Run("failed build step is build_failed regardless of message", func(t *testing.T) {
		steps := []db.DeploymentStep{
			{Step: db.DeploymentStepsStepBuilding, Error: sql.NullString{Valid: true, String: "opaque depot build output"}},
		}
		f := deriveFailure(db.DeploymentsStatusFailed, steps)
		require.NotNil(t, f)
		require.Equal(t, openapi.DeploymentErrorCodeBuildFailed, f.Code)
		require.Equal(t, "building", f.Step)
	})

	t.Run("failed deploying step is classified by message", func(t *testing.T) {
		steps := []db.DeploymentStep{
			{Step: db.DeploymentStepsStepDeploying, Error: sql.NullString{Valid: true, String: deployfail.MsgNoSchedulableRegions}},
		}
		f := deriveFailure(db.DeploymentsStatusFailed, steps)
		require.NotNil(t, f)
		require.Equal(t, openapi.DeploymentErrorCodeNoSchedulableRegions, f.Code)
		require.Equal(t, "deploying", f.Step)
	})
}
