package clidevice

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
)

// Refresh rotates the refresh token and returns a new access token.
// The user's current WorkOS roles are intersected with the stored permissions.
// Removal fails the refresh and revokes the session. The immediately previous
// refresh token still rotates the session during [refreshReuseGrace]. After
// that window, presenting it revokes the family.
func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (Tokens, error) {
	if err := s.requireEnabled(); err != nil {
		return Tokens{}, err
	}
	token := strings.TrimSpace(req.RefreshToken)
	if token == "" {
		return Tokens{}, unauthenticated("The refresh token is invalid. Run unkey login again.")
	}
	presented := hashRefreshToken(token)
	row, err := db.Query.FindCLISessionByRefreshTokenHash(ctx, s.DB.RW(), presented)
	if err != nil {
		if db.IsNotFound(err) {
			return s.refreshPreviousToken(ctx, req, presented)
		}
		return Tokens{}, err
	}
	return s.rotate(ctx, req, row, presented)
}

func (s *Service) rotate(ctx context.Context, req RefreshRequest, row db.CliSession, currentHash string) (Tokens, error) {
	if row.RevokedAt.Valid || row.ExpiresAt <= s.Clock.Now().UnixMilli() {
		return Tokens{}, unauthenticated("This CLI session is no longer valid. Run unkey login again.")
	}

	workspace, err := db.Query.FindWorkspaceByID(ctx, s.DB.RO(), row.WorkspaceID)
	if err != nil {
		return Tokens{}, err
	}
	if !workspaceUsable(workspace) {
		if err := s.endSession(ctx, row); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, forbidden("This workspace cannot refresh a CLI session.")
	}
	roles, found, err := s.Memberships.ActiveRoles(ctx, workspace.OrgID, row.UserID)
	if err != nil {
		return Tokens{}, err
	}
	if !found {
		if err := s.endSession(ctx, row); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, forbidden("You are no longer a member of this workspace. Run unkey login again.")
	}
	stored, err := decodeStrings(row.Permissions)
	if err != nil {
		return Tokens{}, err
	}
	narrowed, err := intersectPermissions(ctx, workspace.ID, roles, stored)
	if err != nil {
		return Tokens{}, err
	}
	if len(narrowed) == 0 {
		if err := s.endSession(ctx, row); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, forbidden("Your role no longer includes the permissions on this CLI session. Run unkey login again.")
	}

	next := newRefreshToken()
	now := s.Clock.Now().UnixMilli()
	permsJSON, err := json.Marshal(narrowed)
	if err != nil {
		return Tokens{}, err
	}
	accessToken, err := s.signAccessToken(row.UserID, row.UserName.String, workspace.ID, row.ID, row.DeviceName.String, narrowed)
	if err != nil {
		return Tokens{}, err
	}

	var tokens Tokens
	err = db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		n, err := db.Query.RotateCLISessionRefreshToken(ctx, tx, db.RotateCLISessionRefreshTokenParams{
			RefreshTokenHash:         hashRefreshToken(next),
			PreviousRefreshTokenHash: sql.NullString{String: currentHash, Valid: true},
			RotatedAt:                sql.NullInt64{Int64: now, Valid: true},
			LastUsedAt:               now,
			ExpiresAt:                sessionExpiry(row.CreatedAt, now),
			Permissions:              permsJSON,
			ID:                       row.ID,
			CurrentRefreshTokenHash:  currentHash,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return unauthenticated("This refresh token was already used. Run unkey login again.")
		}
		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		if err := s.writeAudit(ctx, tx, auditEntry{
			Event:       auditlog.CLISessionRefreshEvent,
			Display:     "Refreshed CLI session " + row.ID,
			UserID:      row.UserID,
			UserName:    row.UserName.String,
			WorkspaceID: workspace.ID,
			SessionID:   row.ID,
			DeviceName:  row.DeviceName.String,
			RemoteIP:    req.RemoteIP,
			UserAgent:   req.UserAgent,
		}); err != nil {
			return err
		}
		tokens = Tokens{
			AccessToken:  accessToken,
			RefreshToken: next,
			ExpiresIn:    int64(accessTTL / time.Second),
			WorkspaceID:  workspace.ID,
			SessionID:    row.ID,
		}
		return nil
	})
	return tokens, err
}

func (s *Service) refreshPreviousToken(ctx context.Context, req RefreshRequest, presented string) (Tokens, error) {
	row, err := db.Query.FindCLISessionByPreviousRefreshTokenHash(ctx, s.DB.RW(), sql.NullString{String: presented, Valid: true})
	if err != nil {
		if db.IsNotFound(err) {
			return Tokens{}, unauthenticated("The refresh token is invalid. Run unkey login again.")
		}
		return Tokens{}, err
	}
	if row.RevokedAt.Valid {
		return Tokens{}, unauthenticated("This CLI session is no longer valid. Run unkey login again.")
	}
	if refreshReuseAllowed(row.RotatedAt, s.Clock.Now().UnixMilli()) {
		return s.rotate(ctx, req, row, row.RefreshTokenHash)
	}
	if err := s.endSession(ctx, row); err != nil {
		return Tokens{}, err
	}
	return Tokens{}, unauthenticated("This refresh token was already used. The CLI session was revoked. Run unkey login again.")
}

func refreshReuseAllowed(rotatedAt sql.NullInt64, now int64) bool {
	if !rotatedAt.Valid {
		return false
	}
	age := now - rotatedAt.Int64
	return age >= 0 && age <= refreshReuseGrace.Milliseconds()
}

func (s *Service) endSession(ctx context.Context, row db.CliSession) error {
	if _, err := db.Query.RevokeCLISessionFamily(ctx, s.DB.RW(), db.RevokeCLISessionFamilyParams{
		RevokedAt:     sql.NullInt64{Int64: s.Clock.Now().UnixMilli(), Valid: true},
		RefreshFamily: row.RefreshFamily,
	}); err != nil {
		return err
	}
	s.forget(ctx, row.ID)
	return nil
}
