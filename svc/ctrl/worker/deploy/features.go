package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/unkeyed/unkey/pkg/featureflag"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

func (w *Workflow) decideFeatures(ctx context.Context, deploymentID string, target db.FindDeployTargetRow) (mysqltype.DeploymentFeatures, error) {
	existing, err := w.db.FindDeploymentForCreate(ctx, deploymentID)
	if err == nil {
		return existing.Features, nil
	}
	if !db.IsNotFound(err) {
		return mysqltype.DeploymentFeatures{PrivateNetworking: false}, fmt.Errorf("failed to look up deployment %s: %w", deploymentID, err)
	}

	privateNetworking, err := w.decidePrivateNetworking(ctx, target)
	if err != nil {
		return mysqltype.DeploymentFeatures{PrivateNetworking: false}, err
	}
	return mysqltype.DeploymentFeatures{PrivateNetworking: privateNetworking}, nil
}

func (w *Workflow) decidePrivateNetworking(ctx context.Context, target db.FindDeployTargetRow) (bool, error) {
	if !target.PrivateNetworkEligible {
		return false, nil
	}

	detail, err := w.flags.BooleanValueDetails(ctx, featureflag.PrivateNetworking, false, featureflag.TeamContext(target.WorkspaceOrgID))
	if err != nil {
		result := flagErrorResult(detail.ErrorCode)
		metrics.FeatureFlagEvaluationsTotal.WithLabelValues(featureflag.PrivateNetworking, result).Inc()
		logger.Warn("feature flag evaluation failed",
			"flag", featureflag.PrivateNetworking,
			"result", result,
			"workspace_id", target.WorkspaceID,
		)
		return false, fmt.Errorf("evaluate feature flag %s: %s", featureflag.PrivateNetworking, result)
	}

	result := "disabled"
	if detail.Value {
		result = "enabled"
	}
	metrics.FeatureFlagEvaluationsTotal.WithLabelValues(featureflag.PrivateNetworking, result).Inc()
	return detail.Value, nil
}

func flagErrorResult(code openfeature.ErrorCode) string {
	switch code {
	case openfeature.ProviderNotReadyCode,
		openfeature.ProviderFatalCode,
		openfeature.FlagNotFoundCode,
		openfeature.ParseErrorCode,
		openfeature.TypeMismatchCode,
		openfeature.TargetingKeyMissingCode,
		openfeature.InvalidContextCode:
		return strings.ToLower(string(code))
	case openfeature.GeneralCode:
		return "general"
	default:
		return "general"
	}
}
