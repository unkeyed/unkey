package deploy

import (
	"context"
	"fmt"

	restate "github.com/restatedev/sdk-go"
	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// ConcurrencyLimitKey is the flow control limit key of a Deploy send. With the
// workspace id scope it forms the pattern "<workspace_id>/deploy" that Restate
// matches against its rule book.
const ConcurrencyLimitKey = "deploy"

const workflowServiceName = "hydra.v1.DeployWorkflow"

const defaultDeployConcurrency uint32 = 1

// SendDeploy submits Deploy through ingress scoped to the workspace and
// carrying the concurrency limit key, so Restate queues it behind the
// workspace's other deploys. The generated ingress client cannot set a scope.
func SendDeploy(ctx context.Context, client *restateingress.Client, workspaceID string, req *hydrav1.DeployRequest) (restateingress.SimpleSendResponse, error) {
	return restateingress.WorkflowSend[*hydrav1.DeployRequest](
		client, workflowServiceName, req.GetDeploymentId(), "Deploy", restate.WithScope(workspaceID),
	).Send(ctx, req, restate.WithProtoJSON, restate.WithLimitKey(ConcurrencyLimitKey))
}

// SendNotifyInstancesReady resolves the readiness promise of a running Deploy.
// The scope is part of the workflow instance identity, so it must match the
// one SendDeploy used. It carries no limit key and never queues behind deploys.
func SendNotifyInstancesReady(ctx context.Context, client *restateingress.Client, workspaceID, deploymentID string) error {
	_, err := restateingress.WorkflowSend[*hydrav1.NotifyInstancesReadyRequest](
		client, workflowServiceName, deploymentID, "NotifyInstancesReady", restate.WithScope(workspaceID),
	).Send(ctx, &hydrav1.NotifyInstancesReadyRequest{DeploymentId: deploymentID}, restate.WithProtoJSON)
	return err
}

func (w *Workflow) syncDeployConcurrencyRule(ctx restate.Context, workspaceID string) error {
	if w.restateAdmin == nil {
		return nil
	}
	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		limit, err := w.db.FindBuildsConcurrentMaxByWorkspaceID(runCtx, workspaceID)
		if err != nil && !db.IsNotFound(err) {
			return fmt.Errorf("failed to load workspace build limit: %w", err)
		}
		// Restate rejects a rule with concurrency zero, so a zero row would
		// fail every sync for the workspace instead of writing a limit
		concurrency := max(defaultDeployConcurrency, uint32(limit))
		return w.restateAdmin.UpsertLimitRule(runCtx, workspaceID+"/"+ConcurrencyLimitKey, concurrency)
	}, restate.WithName("sync deploy concurrency rule"), restate.WithMaxRetryAttempts(runMaxAttempts))
}
