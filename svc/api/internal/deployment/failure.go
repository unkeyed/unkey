package deployment

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// deriveFailure builds the structured failure for a failed deployment from its
// recorded steps. Returns nil for any non-failed status.
func deriveFailure(status db.DeploymentsStatus, steps []db.DeploymentStep) *openapi.DeploymentFailure {
	if status != db.DeploymentsStatusFailed {
		return nil
	}

	// The pipeline stops at the first failing step, but scan for the last recorded
	// error so a late overwrite wins.
	var failed *db.DeploymentStep
	for i := range steps {
		if steps[i].Error.Valid && steps[i].Error.String != "" {
			failed = &steps[i]
		}
	}

	if failed == nil {
		return &openapi.DeploymentFailure{
			Code:    openapi.DeploymentFailureCodeUnknown,
			Step:    "",
			Message: "The deployment failed for an unknown reason.",
		}
	}

	return &openapi.DeploymentFailure{
		Code:    classifyFailure(failed.Step, failed.Error.String),
		Step:    string(failed.Step),
		Message: failed.Error.String,
	}
}

// classifyFailure maps a failed step to a stable code. A failure in the build
// step is always a build failure, classified by step because the worker's build
// error message is rewritten across the Restate boundary and is not stable to
// match on. Other steps are classified by their stored message, which matches
// the shared deployfail constants the worker writes so the two sides cannot
// drift. First contained match wins.
func classifyFailure(step db.DeploymentStepsStep, message string) openapi.DeploymentFailureCode {
	if step == db.DeploymentStepsStepBuilding {
		return openapi.DeploymentFailureCodeBuildFailed
	}
	for _, rule := range failureRules {
		if strings.Contains(message, rule.substr) {
			return rule.code
		}
	}
	return openapi.DeploymentFailureCodeUnknown
}

var failureRules = []struct {
	substr string
	code   openapi.DeploymentFailureCode
}{
	{deployfail.MsgNoSchedulableRegions, openapi.DeploymentFailureCodeNoSchedulableRegions},
	{deployfail.MsgCPUQuotaExceeded, openapi.DeploymentFailureCodeCpuQuotaExceeded},
	{deployfail.MsgMemoryQuotaExceeded, openapi.DeploymentFailureCodeMemoryQuotaExceeded},
	{deployfail.MsgStorageQuotaExceeded, openapi.DeploymentFailureCodeStorageQuotaExceeded},
	{deployfail.MsgPortTooLow, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
	{deployfail.MsgPortTooHigh, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
	{deployfail.MsgCPUTooLow, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
	{deployfail.MsgMemoryTooLow, openapi.DeploymentFailureCodeInvalidRuntimeSettings},
}
