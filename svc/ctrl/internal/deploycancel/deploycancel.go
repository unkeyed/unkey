// Package deploycancel is the one implementation of "abort these deployments":
// stamp the user-visible reason onto any in-flight steps, transition the rows
// to their terminal status, then kill the Restate invocations driving them.
// The user cancel RPC, sibling dedup on create, and environment deletion all
// route through [Cancel] so the three paths cannot drift apart.
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

// InvocationCanceler cancels a Restate invocation by id. 404 must count as
// success: the workflow finishing in the gap between lookup and cancel is the
// normal race, not an error. *restateadmin.Client satisfies this.
type InvocationCanceler interface {
	CancelInvocation(ctx context.Context, invocationID string) error
}

// Audit describes the deployment.cancel entries to write, one per target.
// A nil Audit (or nil Actor) writes none: sibling dedup is machine-initiated
// noise the customer never asked for, while user cancels and environment
// deletes carry the caller's actor.
type Audit struct {
	Service       auditlogs.AuditLogService
	Actor         *ctrlv1.ActorInfo
	CorrelationID string
	WorkspaceID   string
	// Meta is attached to every entry's deployment resource. Callers pass the
	// project/app/environment ids; all targets in one call share it.
	Meta map[string]any
}

// Params bundles what varies between the three cancel paths.
type Params struct {
	Targets []Target
	// Reason is stamped onto in-flight deployment steps. First-write-wins
	// (WHERE ended_at IS NULL), so the Deploy handler's own step-end loses and
	// the UI shows why the cancel happened, not what it broke.
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
//     The status flip must land BEFORE the invocation cancel: cancellation
//     triggers Deploy's compensation stack, which sets status=failed through
//     the same guard, so flipping first is what makes the intended status win
//     that race. Both writes are best-effort; a failure is logged and the
//     cancel still fires, because a leaked invocation costs more than a
//     cosmetic status.
//  2. Cancel the invocations. Failures don't stop the loop; they are joined
//     and returned so the caller's retry re-runs the whole call, which is safe
//     because every step tolerates rows already in their end state.
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
