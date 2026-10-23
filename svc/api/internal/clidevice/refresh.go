package clidevice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
)

var errRefreshRotated = errors.New("refresh token rotated concurrently")

// Refresh rotates the refresh token and returns a new access token.
// The user's current WorkOS roles are intersected with the stored permissions.
// Removal fails the refresh and revokes the session. The immediately previous
// refresh token still rotates the session during [refreshReuseGrace]. After
// that window, presenting it or any older token of the family revokes the
// family.
func (s *Service) Refresh(ctx context.Context, req RefreshRequest) (Tokens, error) {
	if err := s.requireEnabled(); err != nil {
		return Tokens{}, err
	}
	token := strings.TrimSpace(req.RefreshToken)
	family, ok := refreshTokenFamily(token)
	if !ok {
		return Tokens{}, invalidRefreshToken()
	}

	presented := hashRefreshToken(token)
	row, err := db.Query.FindCLISessionByRefreshTokenHash(ctx, s.DB.RW(), presented)
	if err != nil {
		if db.IsNotFound(err) {
			return s.refreshPreviousToken(ctx, req, presented, family)
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

	stored, err := decodePermissions(row.Permissions)
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

	next := newRefreshToken(row.RefreshFamily)
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
			RefreshTokenHash: hashRefreshToken(next),
			PreviousRefreshTokenHash: sql.NullString{
				String: currentHash,
				Valid:  true,
			},
			RotatedAt: sql.NullInt64{
				Int64: now,
				Valid: true,
			},
			LastUsedAt:              now,
			ExpiresAt:               sessionExpiry(row.CreatedAt, now),
			Permissions:             permsJSON,
			ID:                      row.ID,
			CurrentRefreshTokenHash: currentHash,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return errRefreshRotated
		}

		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		if err := s.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			Event:         auditlog.CLISessionRefreshEvent,
			Display:       "Refreshed CLI session " + row.ID,
			WorkspaceID:   workspace.ID,
			ActorType:     auditlog.UserActor,
			ActorID:       row.UserID,
			ActorName:     row.UserName.String,
			ActorMeta:     map[string]any{"via": "cli", "session_id": row.ID, "device": row.DeviceName.String},
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

		tokens = Tokens{
			AccessToken:  accessToken,
			RefreshToken: next,
			ExpiresIn:    int64(accessTTL / time.Second),
			WorkspaceID:  workspace.ID,
			SessionID:    row.ID,
		}
		return nil
	})
	if errors.Is(err, errRefreshRotated) {
		return s.refreshPreviousToken(ctx, req, hashRefreshToken(strings.TrimSpace(req.RefreshToken)), row.RefreshFamily)
	}
	return tokens, err
}

func (s *Service) refreshPreviousToken(ctx context.Context, req RefreshRequest, presented string, family string) (Tokens, error) {
	row, err := db.Query.FindCLISessionByPreviousRefreshTokenHash(ctx, s.DB.RW(), sql.NullString{
		String: presented,
		Valid:  true,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return Tokens{}, s.refreshOlderToken(ctx, family)
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
	return Tokens{}, reusedRefreshToken()
}

// refreshOlderToken handles a well-formed token that is neither the current
// nor the previous one. If its family still has a live session, the token was
// rotated away more than once, so it is reuse and the family is revoked.
func (s *Service) refreshOlderToken(ctx context.Context, family string) error {
	row, err := db.Query.FindUnrevokedCLISessionByRefreshFamily(ctx, s.DB.RW(), family)
	if err != nil {
		if db.IsNotFound(err) {
			return invalidRefreshToken()
		}
		return err
	}
	if err := s.endSession(ctx, row); err != nil {
		return err
	}
	return reusedRefreshToken()
}

func invalidRefreshToken() error {
	return unauthenticated("The refresh token is invalid. Run unkey login again.")
}

func reusedRefreshToken() error {
	return unauthenticated("This refresh token was already used. The CLI session was revoked. Run unkey login again.")
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
		RevokedAt: sql.NullInt64{
			Int64: s.Clock.Now().UnixMilli(),
			Valid: true,
		},
		RefreshFamily: row.RefreshFamily,
	}); err != nil {
		return err
	}
	s.SessionCache.Remove(ctx, row.ID)
	return nil
}
