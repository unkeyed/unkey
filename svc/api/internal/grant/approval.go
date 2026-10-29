package grant

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/unkeyed/unkey/pkg/db"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
)

// Get returns the authorization behind a user code.
func (s *Service) Get(ctx context.Context, req GetRequest) (Authorization, error) {
	if _, err := RequireDashboardSession(req.Principal); err != nil {
		return Authorization{}, err
	}
	if err := s.limitUserCodeAttempts(ctx, req.Principal.Subject.ID); err != nil {
		return Authorization{}, err
	}

	row, err := db.Query.FindDeviceAuthorizationByUserCode(ctx, s.DB.RO(), normalizeUserCode(req.UserCode))
	if err != nil {
		if db.IsNotFound(err) {
			return Authorization{}, requestNotFound()
		}
		return Authorization{}, err
	}
	parsed, err := parseRequest(row)
	if err != nil {
		return Authorization{}, err
	}
	return Authorization{
		UserCode:           row.UserCode,
		DeviceName:         row.DeviceName.String,
		RequesterIP:        row.RequesterIp.String,
		RequesterUserAgent: row.RequesterUserAgent.String,
		Status:             string(s.authorizationStatus(row)),
		WorkspaceID:        row.WorkspaceID.String,
		CreatedAt:          row.CreatedAt,
		ExpiresAt:          row.ExpiresAt,
		Client:             parsed.delivery.consentClient(row.DeviceName.String),
	}, nil
}

// Approve stores the permissions the dashboard user granted. For an OAuth
// request it also mints the authorization code, which only this response
// carries.
func (s *Service) Approve(ctx context.Context, req ApproveRequest) (Decided, error) {
	if err := s.requireEnabled(); err != nil {
		return Decided{}, err
	}
	source, err := RequireDashboardSession(req.Principal)
	if err != nil {
		return Decided{}, err
	}
	if err := s.limitUserCodeAttempts(ctx, req.Principal.Subject.ID); err != nil {
		return Decided{}, err
	}

	validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, req.Principal, req.Permissions)
	if err != nil {
		return Decided{}, err
	}

	rolesJSON, err := json.Marshal(source.Roles)
	if err != nil {
		return Decided{}, err
	}
	permsJSON, err := json.Marshal(validated)
	if err != nil {
		return Decided{}, err
	}

	var decided Decided
	err = db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.FindDeviceAuthorizationByUserCodeForUpdate(ctx, tx, normalizeUserCode(req.UserCode))
		if err != nil {
			if db.IsNotFound(err) {
				return requestNotFound()
			}
			return err
		}
		locked, err := parseRequest(row)
		if err != nil {
			return err
		}
		if err := s.rejectClosed(locked); err != nil {
			return err
		}
		if row.ApproverUserID.Valid && row.ApproverUserID.String != req.Principal.Subject.ID {
			return forbidden("Another account already approved this login.")
		}

		workspace, err := db.Query.FindWorkspaceByID(ctx, tx, req.Principal.AuthorizedWorkspaceID)
		if err != nil {
			return err
		}
		if !workspaceUsable(workspace) {
			return forbidden("This workspace cannot approve a " + locked.delivery.holder().loginNoun() + ".")
		}

		now := s.Clock.Now().UnixMilli()
		approved, err := locked.delivery.approve(row.ExpiresAt, now, s.Issuer)
		if err != nil {
			return err
		}
		n, err := db.Query.ApproveDeviceAuthorization(ctx, tx, db.ApproveDeviceAuthorizationParams{
			WorkspaceID: sql.NullString{
				String: req.Principal.AuthorizedWorkspaceID,
				Valid:  true,
			},
			ApproverUserID: sql.NullString{
				String: req.Principal.Subject.ID,
				Valid:  true,
			},
			ApproverName: sql.NullString{
				String: req.Principal.Subject.Name,
				Valid:  req.Principal.Subject.Name != "",
			},
			ApproverRoles: rolesJSON,
			Permissions:   permsJSON,
			ApprovedAt: sql.NullInt64{
				Int64: now,
				Valid: true,
			},
			ExpiresAt: approved.expiresAt,
			CodeHash:  approved.codeHash,
			ID:        row.ID,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return invalid("Could not approve this login. " + locked.delivery.holder().signInAgain(row.DeviceName.String))
		}
		decided = approved.decided
		return nil
	})
	return decided, err
}

// Deny marks the login denied. Denying twice returns the same Decided.
func (s *Service) Deny(ctx context.Context, req DenyRequest) (Decided, error) {
	if err := s.requireEnabled(); err != nil {
		return Decided{}, err
	}
	if _, err := RequireDashboardSession(req.Principal); err != nil {
		return Decided{}, err
	}
	if err := s.limitUserCodeAttempts(ctx, req.Principal.Subject.ID); err != nil {
		return Decided{}, err
	}

	var decided Decided
	err := db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.FindDeviceAuthorizationByUserCodeForUpdate(ctx, tx, normalizeUserCode(req.UserCode))
		if err != nil {
			if db.IsNotFound(err) {
				return requestNotFound()
			}
			return err
		}
		locked, err := parseRequest(row)
		if err != nil {
			return err
		}
		if row.Status == db.DeviceAuthorizationsStatusConsumed {
			return invalid("This login was already used.")
		}
		if row.Status == db.DeviceAuthorizationsStatusDenied {
			decided, err = locked.delivery.deny(s.Issuer)
			return err
		}
		if row.ApproverUserID.Valid && row.ApproverUserID.String != req.Principal.Subject.ID {
			return forbidden("Another account already approved this login.")
		}

		n, err := db.Query.DenyDeviceAuthorization(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		if n != 1 {
			return invalid("This login was already used.")
		}
		decided, err = locked.delivery.deny(s.Issuer)
		return err
	})
	return decided, err
}

func (s *Service) authorizationStatus(row db.DeviceAuthorization) db.DeviceAuthorizationsStatus {
	if row.ExpiresAt <= s.Clock.Now().UnixMilli() &&
		row.Status != db.DeviceAuthorizationsStatusConsumed &&
		row.Status != db.DeviceAuthorizationsStatusDenied {
		return db.DeviceAuthorizationsStatusExpired
	}
	return row.Status
}

func (s *Service) rejectClosed(req request) error {
	holder := req.delivery.holder()
	switch s.authorizationStatus(req.row) {
	case db.DeviceAuthorizationsStatusConsumed:
		return invalid("This login was already used.")
	case db.DeviceAuthorizationsStatusDenied:
		return invalid("This login was denied.")
	case db.DeviceAuthorizationsStatusExpired:
		return invalid("This " + holder.loginNoun() + " expired. " + holder.signInAgain(req.row.DeviceName.String))
	case db.DeviceAuthorizationsStatusPending, db.DeviceAuthorizationsStatusApproved:
		return nil
	default:
		return nil
	}
}

func normalizeUserCode(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}
