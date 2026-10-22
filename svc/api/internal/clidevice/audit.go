package clidevice

import (
	"context"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
)

func (s *Service) writeAudit(ctx context.Context, tx db.DBTX, event auditlog.AuditLogEvent, display string, userID string, name string, workspaceID string, sessionID string, device string, remoteIP string, userAgent string) error {
	actor := auditactor.FromPrincipal(&principal.Principal{
		Version: "",
		Type:    "",
		Subject: principal.Subject{
			ID:   userID,
			Name: name,
			Type: principal.SubjectTypeUser,
		},
		Source: principal.CLISessionSource{
			SessionID:  sessionID,
			DeviceName: device,
		},
		AuthorizedWorkspaceID: workspaceID,
		Permissions:           nil,
	})
	return s.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
		WorkspaceID:   workspaceID,
		Event:         event,
		ActorType:     actor.Type,
		ActorID:       actor.ID,
		ActorName:     actor.Name,
		ActorMeta:     actor.Meta,
		Display:       display,
		RemoteIP:      remoteIP,
		UserAgent:     userAgent,
		CorrelationID: "",
		Targets: []auditlog.AuditLogTarget{{
			Type:        auditlog.CLISessionTargetType,
			ID:          sessionID,
			Name:        device,
			DisplayName: device,
			Meta:        map[string]any{},
		}},
	}})
}
