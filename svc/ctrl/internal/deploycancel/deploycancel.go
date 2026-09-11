// Package deploycancel aborts deployments. The CancelDeployment RPC,
// deploy.Workflow.cancelOlderSiblings, and the environment Delete workflow all go
// through [Cancel].
package deploycancel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Deployment is one deployment to abort. InvocationID is empty when Deploy
// was sent but its id is not on the row yet; the row still transitions and
// Deploy stops itself on the terminal status.
type Deployment struct {
	ID           string
	InvocationID string
}

// Audit describes the deployment.cancel entry written per deployment. Nil, or
// a nil Actor, writes none.
type Audit struct {
	Service       auditlogs.AuditLogService
	Actor         *ctrlv1.ActorInfo
	CorrelationID string
	WorkspaceID   string
	Meta          map[string]any
}

type Params struct {
	Deployments []Deployment
	// Reason is shown on the deployment's open step in the dashboard.
	Reason string
	Status mysqltype.DeploymentsStatus
	Audit  *Audit
}

// Cancel writes Reason on each open step, moves each row to Status, and only
// then cancels the Restate invocations. The order matters: cancelling makes
// Deploy run its compensations, and the one that marks the deployment failed
// only touches progressing rows, so the status written here survives.
//
// Database errors are logged and the invocations are still cancelled; a
// running invocation is worse than a wrong status. A nil admin skips Restate
// and only moves the rows. Audit entries are written last so a retry after a
// cancel error does not duplicate them.
func Cancel(ctx context.Context, database db.Database, admin *restateadmin.Client, p Params) error {
	if len(p.Deployments) == 0 {
		return nil
	}

	now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
	ids := make([]string, 0, len(p.Deployments))
	for _, d := range p.Deployments {
		ids = append(ids, d.ID)
	}

	if err := database.EndActiveDeploymentStepsForDeployments(ctx, db.EndActiveDeploymentStepsForDeploymentsParams{
		EndedAt:       now,
		Error:         sql.NullString{Valid: true, String: p.Reason},
		DeploymentIds: ids,
	}); err != nil {
		logger.Warn("failed to write cancel reason on deployment steps",
			"deployment_ids", ids,
			"reason", p.Reason,
			"error", err,
		)
	}

	if err := database.UpdateDeploymentStatusBatchIfActive(ctx, db.UpdateDeploymentStatusBatchIfActiveParams{
		Status:              p.Status,
		UpdatedAt:           now,
		Ids:                 ids,
		ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
	}); err != nil {
		logger.Warn("failed to update deployment status for cancel",
			"deployment_ids", ids,
			"status", p.Status,
			"error", err,
		)
	}

	var cancelErrs error
	for _, d := range p.Deployments {
		if d.InvocationID == "" || admin == nil {
			continue
		}
		if err := admin.CancelInvocation(ctx, d.InvocationID); err != nil {
			cancelErrs = errors.Join(cancelErrs, fmt.Errorf(
				"cancel invocation %s for deployment %s: %w", d.InvocationID, d.ID, err))
		}
	}
	if cancelErrs != nil {
		return cancelErrs
	}

	if p.Audit == nil || p.Audit.Actor == nil {
		return nil
	}

	entries := make([]auditlog.AuditLog, 0, len(p.Deployments))
	for _, d := range p.Deployments {
		entries = append(entries, auditlog.AuditLog{
			WorkspaceID:   p.Audit.WorkspaceID,
			Event:         auditlog.DeploymentCancelEvent,
			Display:       fmt.Sprintf("Cancelled deployment %s", d.ID),
			ActorID:       p.Audit.Actor.GetId(),
			ActorName:     p.Audit.Actor.GetName(),
			ActorType:     actor.AuditType(p.Audit.Actor.GetType()),
			ActorMeta:     actor.Meta(p.Audit.Actor.GetMeta()),
			RemoteIP:      p.Audit.Actor.GetRemoteIp(),
			UserAgent:     p.Audit.Actor.GetUserAgent(),
			CorrelationID: p.Audit.CorrelationID,
			Resources: []auditlog.AuditLogResource{
				{
					Type:        auditlog.DeploymentResourceType,
					ID:          d.ID,
					Name:        "",
					DisplayName: d.ID,
					Meta:        p.Audit.Meta,
				},
			},
		})
	}
	if err := p.Audit.Service.Insert(ctx, nil, entries); err != nil {
		return fmt.Errorf("insert deployment cancel audit logs: %w", err)
	}

	return nil
}
