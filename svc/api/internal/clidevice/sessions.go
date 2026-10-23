package clidevice

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/rbac"
	rbacpermissions "github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
)

// Allow reports whether a CLI session may authenticate a request.
func (s *Service) Allow(ctx context.Context, sessionID string) (clisession.Session, error) {
	if s.SessionCache != nil {
		if verdict, hit := s.SessionCache.Get(ctx, sessionID); hit == cache.Hit {
			if !verdict.Allowed {
				return clisession.Session{}, clisession.ErrInactive
			}
			return verdict.Session, nil
		}
	}

	row, err := db.Query.FindCLISessionByID(ctx, s.DB.RW(), sessionID)
	if err != nil {
		if db.IsNotFound(err) {
			var inactive clisession.Session
			s.remember(ctx, sessionID, SessionVerdict{
				Allowed: false,
				Session: inactive,
			})
			return clisession.Session{}, clisession.ErrInactive
		}
		return clisession.Session{}, err
	}

	session := clisession.Session{
		UserID:      row.UserID,
		WorkspaceID: row.WorkspaceID,
		DeviceName:  row.DeviceName.String,
		ExpiresAt:   time.UnixMilli(row.ExpiresAt),
	}
	active := !row.RevokedAt.Valid && row.ExpiresAt > s.Clock.Now().UnixMilli()
	if active {
		workspace, err := db.Query.FindWorkspaceByID(ctx, s.DB.RO(), row.WorkspaceID)
		if err != nil && !db.IsNotFound(err) {
			return clisession.Session{}, err
		}
		active = err == nil && workspaceUsable(workspace)
	}

	s.remember(ctx, sessionID, SessionVerdict{
		Allowed: active,
		Session: session,
	})
	if !active {
		return clisession.Session{}, clisession.ErrInactive
	}
	return session, nil
}

func (s *Service) remember(ctx context.Context, sessionID string, verdict SessionVerdict) {
	if s.SessionCache == nil {
		return
	}
	s.SessionCache.Set(ctx, sessionID, verdict)
}

func (s *Service) forget(ctx context.Context, sessionID string) {
	if s.SessionCache == nil {
		return
	}
	s.SessionCache.Remove(ctx, sessionID)
}

// Revoke ends a session. The owner can revoke their own. An admin can revoke
// any session in the workspace.
func (s *Service) Revoke(ctx context.Context, req RevokeRequest) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	if err := RequireUserSession(req.Principal); err != nil {
		return err
	}
	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		return invalid("The session id is invalid.")
	}

	row, err := db.Query.FindCLISessionByID(ctx, s.DB.RW(), sessionID)
	if err != nil {
		if db.IsNotFound(err) {
			return invalid("This CLI session was not found.")
		}
		return err
	}
	if row.WorkspaceID != req.Principal.AuthorizedWorkspaceID {
		return invalid("This CLI session was not found.")
	}
	if row.UserID != req.Principal.Subject.ID && !isAdmin(req.Principal) {
		return forbidden("You can only revoke your own CLI sessions.")
	}
	if row.RevokedAt.Valid {
		s.forget(ctx, row.ID)
		return nil
	}

	return db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		n, err := db.Query.RevokeCLISession(ctx, tx, db.RevokeCLISessionParams{
			RevokedAt: sql.NullInt64{
				Int64: s.Clock.Now().UnixMilli(),
				Valid: true,
			},
			ID:          row.ID,
			WorkspaceID: row.WorkspaceID,
		})
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}

		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		actor := auditactor.FromPrincipal(req.Principal)
		if err := s.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			Event:         auditlog.CLISessionRevokeEvent,
			Display:       "Revoked CLI session " + row.ID,
			WorkspaceID:   row.WorkspaceID,
			ActorType:     actor.Type,
			ActorID:       actor.ID,
			ActorName:     actor.Name,
			ActorMeta:     actor.Meta,
			CorrelationID: "",
			RemoteIP:      req.RemoteIP,
			UserAgent:     req.UserAgent,
			Targets: []auditlog.AuditLogTarget{{
				Type:        auditlog.CLISessionTargetType,
				ID:          row.ID,
				Name:        row.DeviceName.String,
				DisplayName: row.DeviceName.String,
				Meta:        map[string]any{},
			}},
		}}); err != nil {
			return err
		}
		s.forget(ctx, row.ID)
		return nil
	})
}

// List returns the caller's sessions. Admins see every session in the workspace.
func (s *Service) List(ctx context.Context, req ListRequest) ([]SessionInfo, error) {
	if err := RequireUserSession(req.Principal); err != nil {
		return nil, err
	}

	var rows []db.CliSession
	var err error
	now := s.Clock.Now().UnixMilli()
	if isAdmin(req.Principal) {
		rows, err = db.Query.ListCLISessionsByWorkspace(ctx, s.DB.RO(), db.ListCLISessionsByWorkspaceParams{
			WorkspaceID: req.Principal.AuthorizedWorkspaceID,
			Now:         now,
		})
	} else {
		rows, err = db.Query.ListCLISessionsByUser(ctx, s.DB.RO(), db.ListCLISessionsByUserParams{
			WorkspaceID: req.Principal.AuthorizedWorkspaceID,
			UserID:      req.Principal.Subject.ID,
			Now:         now,
		})
	}
	if err != nil {
		return nil, err
	}

	out := make([]SessionInfo, 0, len(rows))
	for _, row := range rows {
		permissions, err := decodeStrings(row.Permissions)
		if err != nil {
			return nil, err
		}
		info := SessionInfo{
			ID:          row.ID,
			UserID:      row.UserID,
			DeviceName:  row.DeviceName.String,
			Permissions: permissions,
			CreatedAt:   row.CreatedAt,
			LastUsedAt:  row.LastUsedAt,
			ExpiresAt:   row.ExpiresAt,
			RevokedAt:   0,
		}
		if row.RevokedAt.Valid {
			info.RevokedAt = row.RevokedAt.Int64
		}
		out = append(out, info)
	}
	return out, nil
}

func isAdmin(p *principal.Principal) bool {
	return p.Authorize(rbac.U(urn.New().Workspace(p.AuthorizedWorkspaceID).RootKey("*"), rbacpermissions.Write)) == nil
}
