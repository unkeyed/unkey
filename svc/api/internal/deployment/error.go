package deployment

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// deriveError builds the structured failure for a failed deployment from its
// recorded steps. Returns nil for any non-failed status.
func deriveError(status db.DeploymentsStatus, steps []db.DeploymentStep) *openapi.DeploymentError {
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
		return &openapi.DeploymentError{
			Code:    openapi.DeploymentErrorCodeUnknown,
			Step:    "",
			Message: "The deployment failed for an unknown reason.",
		}
	}

	return &openapi.DeploymentError{
		Code:    classifyError(failed.Step, failed.Error.String),
		Step:    string(failed.Step),
		Message: failed.Error.String,
	}
}

// classifyError maps a failed step to a stable code. A failure in the build
// step is always a build failure, classified by step because the worker's build
// error message is rewritten across the Restate boundary and is not stable to
// match on. Other steps are classified by their stored message, which matches
// the shared deployfail constants the worker writes so the two sides cannot
// drift. First contained match wins.
func classifyError(step db.DeploymentStepsStep, message string) openapi.DeploymentErrorCode {
	if step == db.DeploymentStepsStepBuilding {
		return openapi.DeploymentErrorCodeBuildFailed
	}
	for _, rule := range errorRules {
		if strings.Contains(message, rule.substr) {
			return rule.code
		}
	}
	return openapi.DeploymentErrorCodeUnknown
}

var errorRules = []struct {
	substr string
	code   openapi.DeploymentErrorCode
}{
	{deployfail.MsgNoSchedulableRegions, openapi.DeploymentErrorCodeNoSchedulableRegions},
	{deployfail.MsgCPUQuotaExceeded, openapi.DeploymentErrorCodeCpuQuotaExceeded},
	{deployfail.MsgMemoryQuotaExceeded, openapi.DeploymentErrorCodeMemoryQuotaExceeded},
	{deployfail.MsgStorageQuotaExceeded, openapi.DeploymentErrorCodeStorageQuotaExceeded},
	{deployfail.MsgPortOutOfRange, openapi.DeploymentErrorCodeInvalidRuntimeSettings},
	{deployfail.MsgCPUTooLow, openapi.DeploymentErrorCodeInvalidRuntimeSettings},
	{deployfail.MsgMemoryTooLow, openapi.DeploymentErrorCodeInvalidRuntimeSettings},
}
