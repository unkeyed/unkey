package clidevice

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/db"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
)

// Get returns the authorization behind a user code.
func (s *Service) Get(ctx context.Context, req GetRequest) (Authorization, error) {
	if err := RequireDashboardSession(req.Principal); err != nil {
		return Authorization{}, err
	}
	code, err := normalizeUserCode(req.UserCode)
	if err != nil {
		return Authorization{}, err
	}
	row, err := db.Query.FindDeviceAuthorizationByUserCode(ctx, s.DB.RO(), code)
	if err != nil {
		if db.IsNotFound(err) {
			return Authorization{}, loginNotFound()
		}
		return Authorization{}, err
	}
	return authorizationFrom(s.present(row)), nil
}

// Approve stores the permissions the dashboard user granted.
// The handler checks the subset first. This checks it again so a caller of
// the service cannot skip it.
func (s *Service) Approve(ctx context.Context, req ApproveRequest) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	if err := RequireDashboardSession(req.Principal); err != nil {
		return err
	}
	code, err := normalizeUserCode(req.UserCode)
	if err != nil {
		return err
	}
	validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, req.Principal, req.Permissions)
	if err != nil {
		return err
	}
	if len(validated) == 0 {
		return invalid("Select at least one permission.")
	}
	source, ok := req.Principal.Source.(principal.JWTSource)
	if !ok {
		return dashboardSessionRequired()
	}
	rolesJSON, err := json.Marshal(source.Roles)
	if err != nil {
		return err
	}
	permsJSON, err := json.Marshal(validated)
	if err != nil {
		return err
	}

	return db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.FindDeviceAuthorizationByUserCodeForUpdate(ctx, tx, code)
		if err != nil {
			if db.IsNotFound(err) {
				return loginNotFound()
			}
			return err
		}
		if err := s.rejectClosed(row); err != nil {
			return err
		}
		if row.ApproverUserID.Valid && row.ApproverUserID.String != req.Principal.Subject.ID {
			return forbidden("Another account already approved this login.")
		}
		workspace, err := db.Query.FindWorkspaceByID(ctx, tx, req.Principal.AuthorizedWorkspaceID)
		if err != nil {
			return err
		}
		if workspace.DeletedAtM.Valid || !workspace.Enabled || strings.TrimSpace(workspace.OrgID) == "" {
			return forbidden("This workspace cannot approve a CLI login.")
		}
		n, err := db.Query.ApproveDeviceAuthorization(ctx, tx, db.ApproveDeviceAuthorizationParams{
			WorkspaceID:    sql.NullString{String: req.Principal.AuthorizedWorkspaceID, Valid: true},
			ApproverUserID: sql.NullString{String: req.Principal.Subject.ID, Valid: true},
			ApproverName:   sql.NullString{String: req.Principal.Subject.Name, Valid: req.Principal.Subject.Name != ""},
			ApproverRoles:  rolesJSON,
			Permissions:    permsJSON,
			ApprovedAt:     sql.NullInt64{Int64: s.Clock.Now().UnixMilli(), Valid: true},
			ID:             row.ID,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return invalid("Could not approve this login. Run unkey login again.")
		}
		return nil
	})
}

// Deny marks the login denied.
func (s *Service) Deny(ctx context.Context, req DenyRequest) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	if err := RequireDashboardSession(req.Principal); err != nil {
		return err
	}
	code, err := normalizeUserCode(req.UserCode)
	if err != nil {
		return err
	}
	return db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.FindDeviceAuthorizationByUserCodeForUpdate(ctx, tx, code)
		if err != nil {
			if db.IsNotFound(err) {
				return loginNotFound()
			}
			return err
		}
		if row.Status == db.DeviceAuthorizationsStatusConsumed {
			return invalid("This login was already used.")
		}
		if row.Status == db.DeviceAuthorizationsStatusDenied {
			return nil
		}
		if row.ApproverUserID.Valid && row.ApproverUserID.String != req.Principal.Subject.ID {
			return forbidden("Another account already approved this login.")
		}
		return db.Query.UpdateDeviceAuthorizationStatus(ctx, tx, db.UpdateDeviceAuthorizationStatusParams{
			Status: db.DeviceAuthorizationsStatusDenied,
			ID:     row.ID,
		})
	})
}

func (s *Service) present(row db.DeviceAuthorization) db.DeviceAuthorization {
	if row.ExpiresAt <= s.Clock.Now().UnixMilli() &&
		row.Status != db.DeviceAuthorizationsStatusConsumed &&
		row.Status != db.DeviceAuthorizationsStatusDenied {
		row.Status = db.DeviceAuthorizationsStatusExpired
	}
	return row
}

func (s *Service) rejectClosed(row db.DeviceAuthorization) error {
	row = s.present(row)
	switch row.Status {
	case db.DeviceAuthorizationsStatusConsumed:
		return invalid("This login was already used.")
	case db.DeviceAuthorizationsStatusDenied:
		return invalid("This login was denied.")
	case db.DeviceAuthorizationsStatusExpired:
		return invalid("This login code expired. Run unkey login again.")
	default:
		return nil
	}
}

func authorizationFrom(row db.DeviceAuthorization) Authorization {
	return Authorization{
		UserCode:           row.UserCode,
		DeviceName:         row.DeviceName.String,
		RequesterIP:        row.RequesterIp.String,
		RequesterUserAgent: row.RequesterUserAgent.String,
		Status:             string(row.Status),
		WorkspaceID:        row.WorkspaceID.String,
		CreatedAt:          row.CreatedAt,
		ExpiresAt:          row.ExpiresAt,
	}
}
