// Package deploycancel aborts deployments: it stamps the user-visible reason
// onto any in-flight steps, transitions the rows to their terminal status, then
// kills the Restate invocations driving them. The user cancel RPC, sibling
// dedup on create, and environment deletion all go through [Cancel].
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
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Target is one deployment to abort: its row id plus the Restate invocation
// driving it. An empty InvocationID means the workflow was never dispatched
// (or the id has not been persisted yet); the row still transitions.
type Target struct {
	ID           string
	InvocationID string
}

// InvocationCanceler cancels a Restate invocation by id. An implementation must
// count 404 as success: the workflow can finish in the gap between lookup and
// cancel. *restateadmin.Client satisfies this.
type InvocationCanceler interface {
	CancelInvocation(ctx context.Context, invocationID string) error
}

// Audit describes the deployment.cancel entries to write, one per target. A nil
// Audit, or a nil Actor, writes none: a machine-initiated cancel has no actor
// to attribute and does not belong in the customer's feed.
type Audit struct {
	Service       auditlogs.AuditLogService
	Actor         *ctrlv1.ActorInfo
	CorrelationID string
	WorkspaceID   string
	// Meta is attached to every entry's deployment resource. All targets in one
	// call share it.
	Meta map[string]any
}

// Params describes one [Cancel] call.
type Params struct {
	Targets []Target
	// Reason is stamped onto in-flight deployment steps. The write is
	// first-write-wins (WHERE ended_at IS NULL), so the Deploy handler's own
	// step-end loses and the UI shows the cancel.
	Reason string
	// Status is the terminal status the rows transition to: cancelled for user
	// and environment cancels, superseded for sibling dedup.
	Status mysqltype.DeploymentsStatus
	Audit  *Audit
}

// Cancel aborts every target. The order is load-bearing:
//
//  1. Stamp the reason onto active steps, then transition the rows, both
//     guarded so a row that already reached a terminal status is left alone.
//     The status flip must land before the invocation cancel: cancelling
//     triggers Deploy's compensation stack, which sets status=failed through
//     the same guard, so flipping first wins that race. Both writes are
//     best-effort; a failure is logged and the cancel still fires, because a
//     leaked invocation costs more than a wrong status.
//  2. Cancel the invocations. A failure does not stop the loop; the errors are
//     joined and returned, so the caller's retry re-runs the whole call. That
//     is safe because every step tolerates rows already in their end state.
//  3. Audit last, only after every invocation is dead, so a retry that still
//     has cancelling to do cannot double-write the entries.
func Cancel(ctx context.Context, database db.Database, admin InvocationCanceler, p Params) error {
	if len(p.Targets) == 0 {
		return nil
	}

	now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
	ids := make([]string, 0, len(p.Targets))
	for _, target := range p.Targets {
		ids = append(ids, target.ID)
	}

	if err := database.EndActiveDeploymentStepsForDeployments(ctx, db.EndActiveDeploymentStepsForDeploymentsParams{
		EndedAt:       now,
		Error:         sql.NullString{Valid: true, String: p.Reason},
		DeploymentIds: ids,
	}); err != nil {
		logger.Warn("failed to stamp cancel reason on deployment steps",
			"deployment_ids", ids,
			"reason", p.Reason,
			"error", err,
		)
	}

	if err := database.UpdateDeploymentStatusBatchIfActive(ctx, db.UpdateDeploymentStatusBatchIfActiveParams{
		Status:           p.Status,
		UpdatedAt:        now,
		Ids:              ids,
		TerminalStatuses: mysqltype.TerminalDeploymentStatuses,
	}); err != nil {
		logger.Warn("failed to transition deployments for cancel",
			"deployment_ids", ids,
			"status", p.Status,
			"error", err,
		)
	}

	var cancelErrs error
	for _, target := range p.Targets {
		if target.InvocationID == "" || admin == nil {
			continue
		}
		if err := admin.CancelInvocation(ctx, target.InvocationID); err != nil {
			cancelErrs = errors.Join(cancelErrs, fmt.Errorf(
				"cancel invocation %s for deployment %s: %w", target.InvocationID, target.ID, err))
		}
	}
	if cancelErrs != nil {
		return cancelErrs
	}

	if p.Audit == nil || p.Audit.Actor == nil {
		return nil
	}

	entries := make([]auditlog.AuditLog, 0, len(p.Targets))
	for _, target := range p.Targets {
		entries = append(entries, auditlog.AuditLog{
			WorkspaceID:   p.Audit.WorkspaceID,
			Event:         auditlog.DeploymentCancelEvent,
			Display:       fmt.Sprintf("Cancelled deployment %s", target.ID),
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
					ID:          target.ID,
					Name:        "",
					DisplayName: target.ID,
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
