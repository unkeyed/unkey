// Package deploycancel aborts deployments: stamp the user-visible reason on the
// in-flight step, move the rows to a terminal status, then kill the Restate
// invocations. The user cancel RPC, sibling dedup, and environment deletion all
// go through [Cancel].
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

// Target is one deployment to abort. An empty InvocationID still transitions
// the row. There is just no invocation to kill.
type Target struct {
	ID           string
	InvocationID string
}

// InvocationCanceler must treat 404 as success: the workflow can finish
// between lookup and cancel.
type InvocationCanceler interface {
	CancelInvocation(ctx context.Context, invocationID string) error
}

// Audit describes the deployment.cancel entry written per target. Nil, or a nil
// Actor, writes none: a machine-initiated cancel has nobody to attribute.
type Audit struct {
	Service       auditlogs.AuditLogService
	Actor         *ctrlv1.ActorInfo
	CorrelationID string
	WorkspaceID   string
	// Meta is attached to every entry's deployment resource.
	Meta map[string]any
}

// Params describes one [Cancel] call.
type Params struct {
	Targets []Target
	// Reason lands on the in-flight step. The write only hits rows with ended_at
	// NULL, so Deploy's own step-end afterwards loses and the UI shows the cancel.
	Reason string
	Status mysqltype.DeploymentsStatus
	Audit  *Audit
}

// Cancel aborts every target. The order matters.
//
// The status flips before the invocation is cancelled. A running Deploy takes
// the cancel as a terminal error and unwinds its compensation stack, which
// writes failed through the same progressing-only guard, so whichever write
// lands first stays. The flip has returned before the cancel is sent, so it is
// always first. A Deploy that has not started yet sees the flipped row at its
// terminal check instead and never builds.
//
// The two row writes are best-effort and only logged, because a leaked
// invocation costs more than a wrong status. Invocation errors are joined and
// returned so the caller retries the whole call, which is safe because every
// write tolerates a row already in its end state. Audit runs last, so a retry
// that still has cancelling to do cannot write the entries twice.
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
		Status:              p.Status,
		UpdatedAt:           now,
		Ids:                 ids,
		ProgressingStatuses: mysqltype.ProgressingDeploymentStatuses,
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
