// Package deploycancel aborts deployments: write the user-visible reason on the
// open deployment step, move the deployment rows to a terminal status, then
// cancel the Restate invocations running Workflow.Deploy. The CancelDeployment
// RPC, dedup.CancelOlderSiblings, and the environment Delete workflow all go
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
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Target is one deployment to abort. InvocationID is the Restate invocation
// running Workflow.Deploy for it. It is empty when Deploy was sent but its id
// has not been written to deployments.invocation_id yet. An empty InvocationID
// still moves the row to Params.Status. If Deploy is running anyway, the status
// check at the top of Deploy returns early because the status is terminal.
type Target struct {
	ID           string
	InvocationID string
}

// InvocationCanceler cancels a Restate invocation by id. It must treat a 404
// from Restate as success: the deployment can finish on its own between the
// caller reading the invocation id from the row and this call, and Restate
// answers 404 for a completed invocation.
type InvocationCanceler interface {
	CancelInvocation(ctx context.Context, invocationID string) error
}

// Audit describes the deployment.cancel audit log entry written per target.
// Nil, or a nil Actor, writes none: a cancel started by the system, such as
// dedup, has no user to attribute it to.
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
	// Reason is written to deployment_steps.error on each target's step that
	// has ended_at NULL, and ended_at is set at the same time.
	// Workflow.DeploymentStep later runs EndDeploymentStep for that same step,
	// but that query also only updates a row with ended_at NULL, so it changes
	// nothing and the dashboard keeps showing Reason.
	Reason string
	Status mysqltype.DeploymentsStatus
	Audit  *Audit
}

// Cancel aborts every target in this order: EndActiveDeploymentStepsForDeployments
// writes Reason on each target's open step, UpdateDeploymentStatusBatchIfActive
// moves each row to Status, then admin.CancelInvocation cancels each Restate
// invocation. The order matters.
//
// When Restate cancels a running Workflow.Deploy, Deploy gets a terminal error
// and runs its deferred compensations. One of them calls
// UpdateDeploymentStatusIfActive to set the status to failed. That query and
// UpdateDeploymentStatusBatchIfActive both change a row only if its status is
// one of mysqltype.ProgressingDeploymentStatuses. The status is already Status
// when CancelInvocation is called, so the failed write finds a non-progressing
// row and changes nothing. A target with an empty InvocationID is not cancelled
// in Restate at all. Its Deploy stops itself instead, because Deploy checks for
// a terminal status before it builds.
//
// The two database writes only log their errors. A Restate invocation left
// running is worse than a row with the wrong status. CancelInvocation errors
// are joined and returned. Cancel is safe to call again: both database writes
// skip rows that already have ended_at or a terminal status, and
// CancelInvocation treats 404 as success. Audit entries are written last so a
// retry does not write them twice.
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
