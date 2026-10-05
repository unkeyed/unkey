package clidevice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	rbacpermissions "github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	principalpermissions "github.com/unkeyed/unkey/svc/api/internal/principal"
	"github.com/unkeyed/unkey/svc/api/internal/rootkey"
)

const (
	statusPending  = "pending"
	statusApproved = "approved"
	statusConsumed = "consumed"
	statusDenied   = "denied"
	statusExpired  = "expired"
)

type PollStatus string

const (
	PollPending             PollStatus = "authorization_pending"
	PollSlowDown            PollStatus = "slow_down"
	PollPermissionsRequired PollStatus = "permissions_required"
	PollDenied              PollStatus = "access_denied"
	PollExpired             PollStatus = "expired_token"
	PollComplete            PollStatus = "complete"
)

type DeviceAuthorizer interface {
	CreateDevice(ctx context.Context, clientID string) (workos.DeviceCode, workos.DeviceAuthorization, error)
}

type Service struct {
	DB               db.Database
	Keys             keys.KeyService
	Auditlogs        auditlogs.AuditLogService
	Clock            clock.Clock
	Devices          DeviceAuthorizer
	ClientID         string
	DashboardBaseURL string
}

func (s *Service) Enabled() bool {
	return s != nil && authorizerReady(s.Devices) && strings.TrimSpace(s.ClientID) != "" && strings.TrimSpace(s.DashboardBaseURL) != ""
}

func authorizerReady(devices DeviceAuthorizer) bool {
	if devices == nil {
		return false
	}
	value := reflect.ValueOf(devices)
	return value.Kind() != reflect.Pointer || !value.IsNil()
}

type Started struct {
	LoginID                 string
	UserCode                string
	VerificationURI         string
	VerificationURIComplete string
	ExpiresIn               int32
	Interval                int32
}

type LoginView struct {
	UserCode              string
	DeviceName            string
	Status                string
	ExpiresAt             int64
	WorkOSVerificationURI string
}

type ApproveRequest struct {
	Principal   *principal.Principal
	UserCode    string
	Name        string
	Permissions []string
}

type PollResult struct {
	Status   PollStatus
	Interval int32
	Detail   string
	KeyID    string
	Key      *string
}

func (s *Service) Start(ctx context.Context, deviceName string) (Started, error) {
	if err := s.requireEnabled(); err != nil {
		return Started{}, err
	}
	deviceName = strings.TrimSpace(deviceName)
	if len(deviceName) > 512 {
		deviceName = deviceName[:512]
	}
	deviceCode, authz, err := s.Devices.CreateDevice(ctx, s.ClientID)
	if err != nil {
		return Started{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("workos device authorization failed"),
			fault.Public("Could not start CLI login. Try again."),
		)
	}
	userCode := strings.ToUpper(strings.TrimSpace(authz.UserCode))
	verificationURI := strings.TrimRight(s.DashboardBaseURL, "/")
	complete, err := dashboardVerificationURL(verificationURI, userCode)
	if err != nil {
		return Started{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("cli dashboard url is invalid"),
			fault.Public("CLI login is not configured on this API."),
		)
	}
	loginID := "cdl_" + uid.Secure(32)
	now := s.Clock.Now().UnixMilli()
	err = db.Query.InsertCLIDeviceLogin(ctx, s.DB.RW(), db.InsertCLIDeviceLoginParams{
		ID:                    loginID,
		UserCode:              userCode,
		DeviceCode:            string(deviceCode),
		WorkosVerificationUri: authz.VerificationURIComplete,
		PollIntervalSeconds:   int32(authz.Interval),
		ExpiresAt:             now + int64(authz.ExpiresIn)*1000,
		Status:                statusPending,
		DeviceName:            sql.NullString{String: deviceName, Valid: deviceName != ""},
		CreatedAt:             now,
	})
	if err != nil {
		return Started{}, err
	}
	return Started{
		LoginID:                 loginID,
		UserCode:                userCode,
		VerificationURI:         verificationURI + "/cli/device",
		VerificationURIComplete: complete,
		ExpiresIn:               int32(authz.ExpiresIn),
		Interval:                int32(authz.Interval),
	}, nil
}

func (s *Service) Get(ctx context.Context, userCode string) (LoginView, error) {
	code, err := normalizeUserCode(userCode)
	if err != nil {
		return LoginView{}, err
	}
	row, err := db.Query.FindCLIDeviceLoginByUserCode(ctx, s.DB.RO(), code)
	if err != nil {
		if db.IsNotFound(err) {
			return LoginView{}, loginNotFound()
		}
		return LoginView{}, err
	}
	if s.expired(row) && row.Status != statusConsumed {
		row.Status = statusExpired
	}
	return loginView(row), nil
}

func (s *Service) Approve(ctx context.Context, req ApproveRequest) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	code, err := normalizeUserCode(req.UserCode)
	if err != nil {
		return err
	}
	source, ok := req.Principal.Source.(principal.JWTSource)
	if !ok || len(source.Roles) == 0 {
		return fault.New("cli login approval requires a dashboard session",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Public("Sign in to the dashboard to approve this login."),
		)
	}
	if err := req.Principal.Authorize(rbac.U(urn.New().Workspace(req.Principal.AuthorizedWorkspaceID).RootKey("*"), rbacpermissions.Write)); err != nil {
		return err
	}
	validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, req.Principal, req.Permissions)
	if err != nil {
		return err
	}
	if len(validated) == 0 {
		return invalid("Select at least one permission.")
	}
	rolesJSON, err := json.Marshal(source.Roles)
	if err != nil {
		return err
	}
	permsJSON, err := json.Marshal(validated)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(req.Name)
	return db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.FindCLIDeviceLoginByUserCodeForUpdate(ctx, tx, code)
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
			return fault.New("cli login approved by someone else",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Public("Another account already approved this login."),
			)
		}
		workspace, err := db.Query.FindWorkspaceByID(ctx, tx, req.Principal.AuthorizedWorkspaceID)
		if err != nil {
			return err
		}
		if workspace.DeletedAtM.Valid || !workspace.Enabled || strings.TrimSpace(workspace.OrgID) == "" {
			return fault.New("cli login workspace is not usable",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Public("This workspace cannot approve a CLI login."),
			)
		}
		n, err := db.Query.ApproveCLIDeviceLogin(ctx, tx, db.ApproveCLIDeviceLoginParams{
			WorkspaceID:    sql.NullString{String: req.Principal.AuthorizedWorkspaceID, Valid: true},
			ApproverUserID: sql.NullString{String: req.Principal.Subject.ID, Valid: true},
			ApproverName:   sql.NullString{String: req.Principal.Subject.Name, Valid: req.Principal.Subject.Name != ""},
			ApproverRoles:  rolesJSON,
			Permissions:    permsJSON,
			KeyName:        sql.NullString{String: name, Valid: name != ""},
			ApprovedAt:     sql.NullInt64{Int64: s.Clock.Now().UnixMilli(), Valid: true},
			ID:             row.ID,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return fault.New("cli login approval did not apply",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Public("Could not approve this login. Run unkey login again."),
			)
		}
		return nil
	})
}

func (s *Service) Deny(ctx context.Context, approver *principal.Principal, userCode string) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	code, err := normalizeUserCode(userCode)
	if err != nil {
		return err
	}
	return db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		row, err := db.Query.FindCLIDeviceLoginByUserCodeForUpdate(ctx, tx, code)
		if err != nil {
			if db.IsNotFound(err) {
				return loginNotFound()
			}
			return err
		}
		if row.Status == statusConsumed {
			return invalid("This login was already used.")
		}
		if row.Status == statusDenied {
			return nil
		}
		if row.ApproverUserID.Valid && row.ApproverUserID.String != approver.Subject.ID {
			return fault.New("cli login deny by someone else",
				fault.Code(codes.Auth.Authorization.Forbidden.URN()),
				fault.Public("Another account already approved this login."),
			)
		}
		return db.Query.UpdateCLIDeviceLoginStatus(ctx, tx, db.UpdateCLIDeviceLoginStatusParams{
			Status: statusDenied,
			ID:     row.ID,
		})
	})
}

func (s *Service) Poll(ctx context.Context, loginID string, remoteIP string, userAgent string) (PollResult, error) {
	if err := s.requireEnabled(); err != nil {
		return PollResult{}, err
	}
	id, err := normalizeLoginID(loginID)
	if err != nil {
		return PollResult{}, err
	}
	row, err := db.Query.FindCLIDeviceLoginByID(ctx, s.DB.RO(), id)
	if err != nil {
		if db.IsNotFound(err) {
			return PollResult{}, loginNotFound()
		}
		return PollResult{}, err
	}
	if row.Status == statusConsumed {
		return PollResult{}, invalid("This login was already used.")
	}
	if closed, ok := s.closedPoll(ctx, row); ok {
		return closed, nil
	}
	if row.Status != statusApproved || !row.ApproverUserID.Valid {
		return PollResult{Status: PollPermissionsRequired, Interval: row.PollIntervalSeconds}, nil
	}
	return s.issue(ctx, row, remoteIP, userAgent)
}

func (s *Service) issue(ctx context.Context, row db.CliDeviceLogin, remoteIP string, userAgent string) (PollResult, error) {
	var result PollResult
	err := db.TxRetry(ctx, s.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		locked, err := db.Query.FindCLIDeviceLoginByIDForUpdate(ctx, tx, row.ID)
		if err != nil {
			return err
		}
		if locked.Status == statusConsumed {
			return invalid("This login was already used.")
		}
		if locked.Status != statusApproved || !locked.ApproverUserID.Valid {
			result = PollResult{Status: PollPermissionsRequired, Interval: locked.PollIntervalSeconds}
			return nil
		}
		workspace, err := db.Query.FindWorkspaceByID(ctx, tx, locked.WorkspaceID.String)
		if err != nil {
			return err
		}
		if workspace.DeletedAtM.Valid || !workspace.Enabled {
			if err := db.Query.UpdateCLIDeviceLoginStatus(ctx, tx, db.UpdateCLIDeviceLoginStatusParams{Status: statusDenied, ID: locked.ID}); err != nil {
				return err
			}
			result = PollResult{Status: PollDenied, Interval: locked.PollIntervalSeconds, Detail: "This workspace cannot approve a CLI login."}
			return nil
		}
		roles, err := decodeStrings(locked.ApproverRoles)
		if err != nil {
			return err
		}
		requested, err := decodeStrings(locked.Permissions)
		if err != nil {
			return err
		}
		approver := &principal.Principal{
			Type: principal.TypeJWT,
			Subject: principal.Subject{
				ID:   locked.ApproverUserID.String,
				Name: locked.ApproverName.String,
				Type: principal.SubjectTypeUser,
			},
			Source:                principal.JWTSource{Roles: roles},
			AuthorizedWorkspaceID: locked.WorkspaceID.String,
			Permissions:           workos.PermissionsForRoles(locked.WorkspaceID.String, roles),
		}
		if err := approver.Authorize(rbac.U(urn.New().Workspace(approver.AuthorizedWorkspaceID).RootKey("*"), rbacpermissions.Write)); err != nil {
			if err := db.Query.UpdateCLIDeviceLoginStatus(ctx, tx, db.UpdateCLIDeviceLoginStatusParams{Status: statusDenied, ID: locked.ID}); err != nil {
				return err
			}
			result = PollResult{Status: PollDenied, Interval: locked.PollIntervalSeconds, Detail: "You no longer have permission to create this root key."}
			return nil
		}
		validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, approver, requested)
		if err != nil {
			if err := db.Query.UpdateCLIDeviceLoginStatus(ctx, tx, db.UpdateCLIDeviceLoginStatusParams{Status: statusDenied, ID: locked.ID}); err != nil {
				return err
			}
			result = PollResult{Status: PollDenied, Interval: locked.PollIntervalSeconds, Detail: fault.UserFacingMessage(err)}
			return nil
		}
		key, err := s.Keys.CreateKeyV1(ctx, keys.CreateKeyV1Request{Prefix: "unkey"})
		if err != nil {
			return err
		}
		var name *string
		if locked.KeyName.Valid {
			name = &locked.KeyName.String
		}
		ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())
		created, err := rootkey.Insert(ctx, tx, rootkey.InsertRequest{
			Principal:   approver,
			Name:        name,
			Permissions: validated,
			Expires:     sql.NullInt64{},
			Key:         key,
			RemoteIP:    remoteIP,
			UserAgent:   userAgent,
			Auditlogs:   s.Auditlogs,
			Clock:       s.Clock,
		})
		if err != nil {
			return err
		}
		n, err := db.Query.ConsumeCLIDeviceLogin(ctx, tx, locked.ID)
		if err != nil {
			return err
		}
		if n != 1 {
			return fault.New("cli login was not consumed", fault.Code(codes.App.Internal.UnexpectedError.URN()))
		}
		secret := created.Key
		result = PollResult{
			Status:   PollComplete,
			Interval: locked.PollIntervalSeconds,
			KeyID:    created.KeyID,
			Key:      &secret,
		}
		return nil
	})
	return result, err
}

func (s *Service) closedPoll(ctx context.Context, row db.CliDeviceLogin) (PollResult, bool) {
	if s.expired(row) && row.Status != statusConsumed && row.Status != statusDenied {
		_ = s.markStatus(ctx, row.ID, statusExpired)
		return PollResult{Status: PollExpired, Interval: row.PollIntervalSeconds, Detail: "The login code expired. Run unkey login again."}, true
	}
	switch row.Status {
	case statusDenied:
		return PollResult{Status: PollDenied, Interval: row.PollIntervalSeconds, Detail: "Authorization was denied."}, true
	case statusExpired:
		return PollResult{Status: PollExpired, Interval: row.PollIntervalSeconds, Detail: "The login code expired. Run unkey login again."}, true
	case statusConsumed:
		return PollResult{}, false
	default:
		return PollResult{}, false
	}
}

func (s *Service) rejectClosed(row db.CliDeviceLogin) error {
	if s.expired(row) {
		return invalid("This login code expired. Run unkey login again.")
	}
	switch row.Status {
	case statusConsumed:
		return invalid("This login was already used.")
	case statusDenied:
		return invalid("This login was denied.")
	case statusExpired:
		return invalid("This login code expired. Run unkey login again.")
	default:
		return nil
	}
}

func (s *Service) expired(row db.CliDeviceLogin) bool {
	return row.ExpiresAt <= s.Clock.Now().UnixMilli()
}

func (s *Service) markStatus(ctx context.Context, id string, status string) error {
	return db.Query.UpdateCLIDeviceLoginStatus(ctx, s.DB.RW(), db.UpdateCLIDeviceLoginStatusParams{Status: status, ID: id})
}

func (s *Service) requireEnabled() error {
	if !s.Enabled() {
		return fault.New("cli auth is not configured",
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Public("CLI login is not configured on this API."),
		)
	}
	return nil
}

func loginView(row db.CliDeviceLogin) LoginView {
	return LoginView{
		UserCode:              row.UserCode,
		DeviceName:            row.DeviceName.String,
		Status:                row.Status,
		ExpiresAt:             row.ExpiresAt,
		WorkOSVerificationURI: row.WorkosVerificationUri,
	}
}

func normalizeUserCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if len(code) < 4 || len(code) > 32 {
		return "", invalid("Enter the code shown in the terminal.")
	}
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '-' {
			return "", invalid("Enter the code shown in the terminal.")
		}
	}
	return code, nil
}

func normalizeLoginID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if len(id) < 8 || len(id) > 48 {
		return "", invalid("The login id is invalid.")
	}
	for _, r := range id {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return "", invalid("The login id is invalid.")
		}
	}
	return id, nil
}

func dashboardVerificationURL(base string, userCode string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("dashboard base url must be an absolute http(s) url")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/cli/device"
	query := parsed.Query()
	query.Set("user_code", userCode)
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""
	return parsed.String(), nil
}

func decodeStrings(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func invalid(message string) error {
	return fault.New(message,
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Public(message),
	)
}

func loginNotFound() error {
	return invalid("This login code was not found. Run unkey login again.")
}
