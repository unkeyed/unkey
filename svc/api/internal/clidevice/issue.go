package clidevice

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

// Poll returns the login state. The first poll after approval creates the
// session and returns tokens once.
func (s *Service) Poll(ctx context.Context, req PollRequest) (PollResult, error) {
	if err := s.requireEnabled(); err != nil {
		return PollResult{}, err
	}
	if err := s.limit(ctx, "poll", req.RemoteIP, pollLimitPerMinute); err != nil {
		return PollResult{}, err
	}

	row, err := db.Query.FindDeviceAuthorizationByID(ctx, s.DB.RW(), strings.TrimSpace(req.LoginID))
	if err != nil {
		if db.IsNotFound(err) {
			return PollResult{}, loginNotFound()
		}
		return PollResult{}, err
	}

	switch s.authorizationStatus(row) {
	case db.DeviceAuthorizationsStatusDenied:
		return PollResult{
			Status:   PollDenied,
			Interval: pollIntervalSeconds,
			Detail:   "Authorization was denied.",
			Tokens:   Tokens{AccessToken: "", RefreshToken: "", ExpiresIn: 0, WorkspaceID: "", SessionID: ""},
		}, nil
	case db.DeviceAuthorizationsStatusExpired:
		return PollResult{
			Status:   PollExpired,
			Interval: pollIntervalSeconds,
			Detail:   "The login code expired. Run unkey login again.",
			Tokens:   Tokens{AccessToken: "", RefreshToken: "", ExpiresIn: 0, WorkspaceID: "", SessionID: ""},
		}, nil
	case db.DeviceAuthorizationsStatusApproved:
		if row.ApproverUserID.Valid && row.WorkspaceID.Valid {
			return s.issue(ctx, row, req.RemoteIP, req.UserAgent)
		}
	case db.DeviceAuthorizationsStatusConsumed:
		return PollResult{}, alreadyIssued()
	case db.DeviceAuthorizationsStatusPending:
	}
	return PollResult{
		Status:   PollPermissionsRequired,
		Interval: pollIntervalSeconds,
		Detail:   "",
		Tokens:   Tokens{AccessToken: "", RefreshToken: "", ExpiresIn: 0, WorkspaceID: "", SessionID: ""},
	}, nil
}

func (s *Service) issue(ctx context.Context, row db.DeviceAuthorization, remoteIP string, userAgent string) (PollResult, error) {
	workspace, err := db.Query.FindWorkspaceByID(ctx, s.DB.RO(), row.WorkspaceID.String)
	if err != nil {
		return PollResult{}, err
	}
	if !workspaceUsable(workspace) {
		if err := s.denyAuthorization(ctx, row.ID); err != nil {
			return PollResult{}, err
		}
		return PollResult{
			Status:   PollDenied,
			Interval: pollIntervalSeconds,
			Detail:   "This workspace cannot approve a CLI login.",
			Tokens:   Tokens{AccessToken: "", RefreshToken: "", ExpiresIn: 0, WorkspaceID: "", SessionID: ""},
		}, nil
	}

	roles, found, err := s.Memberships.ActiveRoles(ctx, workspace.OrgID, row.ApproverUserID.String)
	if err != nil {
		return PollResult{}, err
	}
	requested, err := decodePermissions(row.Permissions)
	if err != nil {
		return PollResult{}, err
	}
	var narrowed []string
	if found {
		narrowed, err = intersectPermissions(ctx, workspace.ID, roles, requested)
		if err != nil {
			return PollResult{}, err
		}
	}
	if len(narrowed) == 0 {
		if err := s.denyAuthorization(ctx, row.ID); err != nil {
			return PollResult{}, err
		}
		return PollResult{
			Status:   PollDenied,
			Interval: pollIntervalSeconds,
			Detail:   "You no longer have permission to finish this login.",
			Tokens:   Tokens{AccessToken: "", RefreshToken: "", ExpiresIn: 0, WorkspaceID: "", SessionID: ""},
		}, nil
	}

	sessionID := uid.New(uid.CLISessionPrefix)
	family := newRefreshFamily()
	refreshToken := newRefreshToken(family)
	now := s.Clock.Now().UnixMilli()
	permsJSON, err := json.Marshal(narrowed)
	if err != nil {
		return PollResult{}, err
	}
	accessToken, err := s.signAccessToken(row.ApproverUserID.String, row.ApproverName.String, workspace.ID, sessionID, row.DeviceName.String, narrowed)
	if err != nil {
		return PollResult{}, err
	}

	var result PollResult
	err = db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		locked, err := db.Query.FindDeviceAuthorizationByIDForUpdate(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		if locked.Status == db.DeviceAuthorizationsStatusConsumed {
			return alreadyIssued()
		}
		if locked.Status != db.DeviceAuthorizationsStatusApproved || approvalChanged(row, locked) {
			result = PollResult{
				Status:   PollPermissionsRequired,
				Interval: pollIntervalSeconds,
				Detail:   "",
				Tokens:   Tokens{AccessToken: "", RefreshToken: "", ExpiresIn: 0, WorkspaceID: "", SessionID: ""},
			}
			return nil
		}

		if err := db.Query.InsertCLISession(ctx, tx, db.InsertCLISessionParams{
			ID:               sessionID,
			WorkspaceID:      workspace.ID,
			UserID:           locked.ApproverUserID.String,
			UserName:         locked.ApproverName,
			Permissions:      permsJSON,
			DeviceName:       locked.DeviceName,
			RefreshTokenHash: hashRefreshToken(refreshToken),
			RefreshFamily:    family,
			CreatedAt:        now,
			LastUsedAt:       now,
			ExpiresAt:        sessionExpiry(now, now),
		}); err != nil {
			return err
		}
		n, err := db.Query.ConsumeDeviceAuthorization(ctx, tx, db.ConsumeDeviceAuthorizationParams{
			SessionID: sql.NullString{
				String: sessionID,
				Valid:  true,
			},
			ID: locked.ID,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return alreadyIssued()
		}

		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		if err := s.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			Event:         auditlog.CLISessionCreateEvent,
			Display:       "Created CLI session " + sessionID,
			WorkspaceID:   workspace.ID,
			ActorType:     auditlog.UserActor,
			ActorID:       locked.ApproverUserID.String,
			ActorName:     locked.ApproverName.String,
			ActorMeta:     map[string]any{},
			CorrelationID: "",
			RemoteIP:      remoteIP,
			UserAgent:     userAgent,
			Targets: []auditlog.AuditLogTarget{{
				Type:        auditlog.CLISessionTargetType,
				ID:          sessionID,
				Name:        locked.DeviceName.String,
				DisplayName: locked.DeviceName.String,
				Meta:        map[string]any{},
			}},
		}}); err != nil {
			return err
		}

		result = PollResult{
			Status:   PollComplete,
			Interval: pollIntervalSeconds,
			Detail:   "",
			Tokens: Tokens{
				AccessToken:  accessToken,
				RefreshToken: refreshToken,
				ExpiresIn:    int64(accessTTL / time.Second),
				WorkspaceID:  workspace.ID,
				SessionID:    sessionID,
			},
		}
		return nil
	})
	return result, err
}

func approvalChanged(read db.DeviceAuthorization, locked db.DeviceAuthorization) bool {
	return read.ApprovedAt != locked.ApprovedAt ||
		read.ApproverUserID != locked.ApproverUserID ||
		read.WorkspaceID != locked.WorkspaceID ||
		!bytes.Equal(read.Permissions, locked.Permissions)
}

func (s *Service) denyAuthorization(ctx context.Context, id string) error {
	n, err := db.Query.DenyDeviceAuthorization(ctx, s.DB.RW(), id)
	if err != nil {
		return err
	}
	if n != 1 {
		return alreadyIssued()
	}
	return nil
}
