package clidevice

import (
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
	id, err := normalizeLoginID(req.LoginID)
	if err != nil {
		return PollResult{}, err
	}
	if err := s.limit(ctx, "poll", req.RemoteIP, pollLimitPerMinute); err != nil {
		return PollResult{}, err
	}
	row, err := db.Query.FindDeviceAuthorizationByID(ctx, s.DB.RW(), id)
	if err != nil {
		if db.IsNotFound(err) {
			return PollResult{}, loginNotFound()
		}
		return PollResult{}, err
	}
	if row.Status == db.DeviceAuthorizationsStatusConsumed {
		return PollResult{}, alreadyIssued()
	}
	row = s.present(row)
	switch row.Status {
	case db.DeviceAuthorizationsStatusDenied:
		return deniedPoll("Authorization was denied."), nil
	case db.DeviceAuthorizationsStatusExpired:
		return expiredPoll(), nil
	case db.DeviceAuthorizationsStatusApproved:
		if !row.ApproverUserID.Valid || !row.WorkspaceID.Valid {
			return waitingPoll(), nil
		}
		return s.issue(ctx, row, req.RemoteIP, req.UserAgent)
	default:
		return waitingPoll(), nil
	}
}

func (s *Service) issue(ctx context.Context, row db.DeviceAuthorization, remoteIP string, userAgent string) (PollResult, error) {
	workspace, err := db.Query.FindWorkspaceByID(ctx, s.DB.RO(), row.WorkspaceID.String)
	if err != nil {
		return PollResult{}, err
	}
	if workspace.DeletedAtM.Valid || !workspace.Enabled || strings.TrimSpace(workspace.OrgID) == "" {
		if err := s.denyAuthorization(ctx, row.ID); err != nil {
			return PollResult{}, err
		}
		return deniedPoll("This workspace cannot approve a CLI login."), nil
	}
	roles, found, err := s.Memberships.ActiveRoles(ctx, workspace.OrgID, row.ApproverUserID.String)
	if err != nil {
		return PollResult{}, err
	}
	requested, err := decodeStrings(row.Permissions)
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
		return deniedPoll("You no longer have permission to finish this login."), nil
	}

	sessionID := uid.New(uid.CLISessionPrefix)
	refreshToken := newRefreshToken()
	family := uid.Secure(20)
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
		if locked.Status != db.DeviceAuthorizationsStatusApproved {
			result = waitingPoll()
			return nil
		}
		if err := db.Query.InsertCLISession(ctx, tx, db.InsertCLISessionParams{
			ID:               sessionID,
			WorkspaceID:      workspace.ID,
			UserID:           locked.ApproverUserID.String,
			Permissions:      permsJSON,
			DeviceName:       locked.DeviceName,
			RefreshTokenHash: hashRefreshToken(refreshToken),
			RefreshFamily:    family,
			CreatedAt:        now,
			LastUsedAt:       now,
			ExpiresAt:        now + refreshIdle.Milliseconds(),
		}); err != nil {
			return err
		}
		n, err := db.Query.ConsumeDeviceAuthorization(ctx, tx, db.ConsumeDeviceAuthorizationParams{
			SessionID: sql.NullString{String: sessionID, Valid: true},
			ID:        locked.ID,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return alreadyIssued()
		}
		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		if err := s.writeAudit(ctx, tx, auditlog.CLISessionCreateEvent, "Created CLI session "+sessionID, locked.ApproverUserID.String, locked.ApproverName.String, workspace.ID, sessionID, locked.DeviceName.String, remoteIP, userAgent); err != nil {
			return err
		}
		result = PollResult{
			Status:       PollComplete,
			Interval:     pollIntervalSeconds,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int64(accessTTL / time.Second),
			WorkspaceID:  workspace.ID,
			SessionID:    sessionID,
		}
		return nil
	})
	return result, err
}

func (s *Service) denyAuthorization(ctx context.Context, id string) error {
	return db.Query.UpdateDeviceAuthorizationStatus(ctx, s.DB.RW(), db.UpdateDeviceAuthorizationStatusParams{
		Status: db.DeviceAuthorizationsStatusDenied,
		ID:     id,
	})
}

func waitingPoll() PollResult {
	return PollResult{Status: PollPermissionsRequired, Interval: pollIntervalSeconds}
}

func deniedPoll(detail string) PollResult {
	return PollResult{Status: PollDenied, Interval: pollIntervalSeconds, Detail: detail}
}

func expiredPoll() PollResult {
	return PollResult{Status: PollExpired, Interval: pollIntervalSeconds, Detail: "The login code expired. Run unkey login again."}
}
