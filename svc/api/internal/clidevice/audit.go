package clidevice

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
)

type auditEntry struct {
	Event       auditlog.AuditLogEvent
	Display     string
	UserID      string
	UserName    string
	WorkspaceID string
	SessionID   string
	DeviceName  string
	RemoteIP    string
	UserAgent   string
}

func (s *Service) writeAudit(ctx context.Context, tx db.DBTX, entry auditEntry) error {
	actor := auditactor.FromPrincipal(&principal.Principal{
		Version: "",
		Type:    "",
		Subject: principal.Subject{
			ID:   entry.UserID,
			Name: entry.UserName,
			Type: principal.SubjectTypeUser,
		},
		Source: principal.CLISessionSource{
			SessionID:  entry.SessionID,
			DeviceName: entry.DeviceName,
			ExpiresAt:  time.Time{},
		},
		AuthorizedWorkspaceID: entry.WorkspaceID,
		Permissions:           nil,
	})

	return s.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
		{
			WorkspaceID:   entry.WorkspaceID,
			Event:         entry.Event,
			ActorType:     actor.Type,
			ActorID:       actor.ID,
			ActorName:     actor.Name,
			ActorMeta:     actor.Meta,
			Display:       entry.Display,
			RemoteIP:      entry.RemoteIP,
			UserAgent:     entry.UserAgent,
			CorrelationID: "",
			Targets: []auditlog.AuditLogTarget{
				{
					Type:        auditlog.CLISessionTargetType,
					ID:          entry.SessionID,
					Name:        entry.DeviceName,
					DisplayName: entry.DeviceName,
					Meta:        map[string]any{},
				},
			},
		},
	})
}
