// Package audit writes audit logs from inside Restate deletion workflows.
//
// Delete RPCs enqueue a durable workflow and can't share a transaction with
// the audit insert, so writing the audit log on the RPC path risks a
// deleting-but-unaudited window if the insert fails. Workflows instead call
// [Insert], which records the event as its own durable step: the nil tx makes
// auditlogs.Insert open its own transaction, and the enclosing RunVoid journals
// the result so it commits exactly once across retries.
package audit

import (
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
)

// Event is a single audit log entry to write from a workflow. The actor fields
// and the durable-step wiring are shared; callers supply only the parts that
// vary per resource.
type Event struct {
	// Actor is the caller threaded in via the workflow request. A cascade
	// (project -> app -> environment) carries the same actor down every level.
	Actor *ctrlv1.ActorInfo

	// CorrelationID groups this event with the other deletions in the same
	// teardown. Minted once at the RPC entry point and threaded down.
	CorrelationID string

	WorkspaceID string
	Event       auditlog.AuditLogEvent
	Display     string
	Resource    auditlog.AuditLogResource
}

// Insert writes e as a durable Restate step named "insert audit log".
func Insert(ctx restate.ObjectContext, svc auditlogs.AuditLogService, e Event) error {
	a := e.Actor
	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return svc.Insert(runCtx, nil, []auditlog.AuditLog{
			{
				WorkspaceID:   e.WorkspaceID,
				Event:         e.Event,
				Display:       e.Display,
				ActorID:       a.GetId(),
				ActorName:     a.GetName(),
				ActorType:     actor.AuditType(a.GetType()),
				ActorMeta:     actor.Meta(a.GetMeta()),
				RemoteIP:      a.GetRemoteIp(),
				UserAgent:     a.GetUserAgent(),
				CorrelationID: e.CorrelationID,
				Resources:     []auditlog.AuditLogResource{e.Resource},
			},
		})
	}, restate.WithName("insert audit log"))
}
