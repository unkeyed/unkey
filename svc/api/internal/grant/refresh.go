package grant

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/clisession"
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
		return Tokens{}, invalidRefreshToken(req.Holder, "")
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
	holder := holderOf(row)
	if holder != req.Holder {
		return Tokens{}, invalidRefreshToken(req.Holder, "")
	}
	label := row.DeviceName.String
	if row.RevokedAt.Valid || row.ExpiresAt <= s.Clock.Now().UnixMilli() {
		return Tokens{}, sessionEnded(holder, label)
	}

	workspace, err := db.Query.FindWorkspaceByID(ctx, s.DB.RO(), row.WorkspaceID)
	if err != nil {
		return Tokens{}, err
	}
	if !workspaceUsable(workspace) {
		if err := s.endSession(ctx, row); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, forbidden("This workspace cannot refresh a " + holder.noun() + ".")
	}

	roles, found, err := s.Memberships.ActiveRoles(ctx, workspace.OrgID, row.UserID)
	if err != nil {
		return Tokens{}, err
	}
	if !found {
		if err := s.endSession(ctx, row); err != nil {
			return Tokens{}, err
		}
		return Tokens{}, forbidden("You are no longer a member of this workspace. " + holder.signInAgain(label))
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
		return Tokens{}, forbidden("Your role no longer includes the permissions on this " + holder.noun() + ". " + holder.signInAgain(label))
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

		audit := holder.audit()
		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		if err := s.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			Event:         audit.refresh,
			Display:       "Refreshed " + audit.display + " " + row.ID,
			WorkspaceID:   workspace.ID,
			ActorType:     auditlog.UserActor,
			ActorID:       row.UserID,
			ActorName:     row.UserName.String,
			ActorMeta:     holder.actorMeta(row.ID, label),
			CorrelationID: "",
			RemoteIP:      req.RemoteIP,
			UserAgent:     req.UserAgent,
			Targets: []auditlog.AuditLogTarget{{
				Type:        audit.target,
				ID:          row.ID,
				Name:        label,
				DisplayName: label,
				Meta:        holder.targetMeta(),
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
			return Tokens{}, s.refreshOlderToken(ctx, req.Holder, family)
		}
		return Tokens{}, err
	}
	if row.RevokedAt.Valid {
		return Tokens{}, sessionEnded(holderOf(row), row.DeviceName.String)
	}
	if refreshReuseAllowed(row.RotatedAt, s.Clock.Now().UnixMilli()) {
		return s.rotate(ctx, req, row, row.RefreshTokenHash)
	}

	if err := s.endSession(ctx, row); err != nil {
		return Tokens{}, err
	}
	return Tokens{}, reusedRefreshToken(holderOf(row), row.DeviceName.String)
}

// refreshOlderToken handles a well-formed token that is neither the current
// nor the previous one. If its family still has a live session, the token was
// rotated away more than once, so it is reuse and the family is revoked.
func (s *Service) refreshOlderToken(ctx context.Context, presenter Holder, family string) error {
	row, err := db.Query.FindUnrevokedCLISessionByRefreshFamily(ctx, s.DB.RW(), family)
	if err != nil {
		if db.IsNotFound(err) {
			return invalidRefreshToken(presenter, "")
		}
		return err
	}
	if err := s.endSession(ctx, row); err != nil {
		return err
	}
	return reusedRefreshToken(holderOf(row), row.DeviceName.String)
}

func invalidRefreshToken(h Holder, label string) error {
	return unauthenticated("The refresh token is invalid. " + h.signInAgain(label))
}

func reusedRefreshToken(h Holder, label string) error {
	return unauthenticated("This refresh token was already used. The " + h.noun() + " was revoked. " + h.signInAgain(label))
}

func sessionEnded(h Holder, label string) error {
	return unauthenticated(clisession.SessionEnded(string(h.clientID), label))
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
