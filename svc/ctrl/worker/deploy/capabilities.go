package deploy

import (
	"context"
	"fmt"
	"strings"

	"github.com/open-feature/go-sdk/openfeature"
	"github.com/unkeyed/unkey/pkg/featureflag"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

func (w *Workflow) decideCapabilities(ctx context.Context, deploymentID string, target db.FindDeployTargetRow) (mysqltype.DeploymentCapabilities, error) {
	existing, err := w.db.FindDeploymentForCreate(ctx, deploymentID)
	if err == nil {
		return existing.Capabilities, nil
	}
	if !db.IsNotFound(err) {
		return mysqltype.DeploymentCapabilities{PrivateNetworking: false}, fmt.Errorf("failed to look up deployment %s: %w", deploymentID, err)
	}

	privateNetworking, err := w.decidePrivateNetworking(ctx, target)
	if err != nil {
		return mysqltype.DeploymentCapabilities{PrivateNetworking: false}, err
	}
	return mysqltype.DeploymentCapabilities{PrivateNetworking: privateNetworking}, nil
}

func (w *Workflow) decidePrivateNetworking(ctx context.Context, target db.FindDeployTargetRow) (bool, error) {
	if !target.PrivateNetworkEligible {
		return false, nil
	}

	detail, err := w.flags.BooleanValueDetails(ctx, featureflag.PrivateNetworking, false, featureflag.TeamContext(target.WorkspaceOrgID))
	if err != nil {
		result := flagErrorResult(detail.ErrorCode)
		metrics.FeatureFlagEvaluationsTotal.WithLabelValues(featureflag.PrivateNetworking, result).Inc()
		return false, fmt.Errorf("evaluate feature flag %s for workspace %s: %s", featureflag.PrivateNetworking, target.WorkspaceID, result)
	}

	result := "disabled"
	if detail.Value {
		result = "enabled"
	}
	metrics.FeatureFlagEvaluationsTotal.WithLabelValues(featureflag.PrivateNetworking, result).Inc()
	return detail.Value, nil
}

// flagErrorResult turns an OpenFeature error code into a metric label. The
// SDK sets codes only from its fixed set, so the label stays bounded.
func flagErrorResult(code openfeature.ErrorCode) string {
	if code == "" {
		return strings.ToLower(string(openfeature.GeneralCode))
	}
	return strings.ToLower(string(code))
}
