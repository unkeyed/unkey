package clidevice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
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
	PollPermissionsRequired PollStatus = "permissions_required"
	PollDenied              PollStatus = "access_denied"
	PollExpired             PollStatus = "expired_token"
	PollComplete            PollStatus = "complete"
)

const (
	ratelimitScope      = "cli_device_login"
	startLimitPerMinute = 20
	pollLimitPerMinute  = 120
	// Expired rows stay readable for a day so the dashboard can still explain
	// what happened to a code before cleanup removes it.
	retainExpiredFor = 24 * time.Hour
)

type DeviceAuthorizer interface {
	CreateDevice(ctx context.Context, clientID string) (workos.DeviceCode, workos.DeviceAuthorization, error)
}

type Service struct {
	DB               db.Database
	Keys             keys.KeyService
	Auditlogs        auditlogs.AuditLogService
	Clock            clock.Clock
	Ratelimit        ratelimit.Service
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

type StartRequest struct {
	DeviceName string
	RemoteIP   string
	UserAgent  string
}

type LoginView struct {
	UserCode           string
	DeviceName         string
	RequesterIP        string
	RequesterUserAgent string
	Status             string
	CreatedAt          int64
	ExpiresAt          int64
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

func (s *Service) Start(ctx context.Context, req StartRequest) (Started, error) {
	if err := s.requireEnabled(); err != nil {
		return Started{}, err
	}
	if err := s.limit(ctx, "start", req.RemoteIP, startLimitPerMinute); err != nil {
		return Started{}, err
	}
	s.deleteExpired(ctx)
	deviceName := truncate(strings.TrimSpace(req.DeviceName), 512)
	userAgent := truncate(strings.TrimSpace(req.UserAgent), 512)
	_, authz, err := s.Devices.CreateDevice(ctx, s.ClientID)
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
		ID:                  loginID,
		UserCode:            userCode,
		PollIntervalSeconds: int32(authz.Interval),
		ExpiresAt:           now + int64(authz.ExpiresIn)*1000,
		Status:              statusPending,
		DeviceName:          sql.NullString{String: deviceName, Valid: deviceName != ""},
		RequesterIp:         sql.NullString{String: req.RemoteIP, Valid: req.RemoteIP != ""},
		RequesterUserAgent:  sql.NullString{String: userAgent, Valid: userAgent != ""},
		CreatedAt:           now,
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

func (s *Service) Get(ctx context.Context, viewer *principal.Principal, userCode string) (LoginView, error) {
	if _, err := dashboardSession(viewer); err != nil {
		return LoginView{}, err
	}
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
	source, err := dashboardSession(req.Principal)
	if err != nil {
		return err
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
	if _, err := dashboardSession(approver); err != nil {
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
	if err := s.limit(ctx, "poll", remoteIP, pollLimitPerMinute); err != nil {
		return PollResult{}, err
	}
	row, err := db.Query.FindCLIDeviceLoginByID(ctx, s.DB.RW(), id)
	if err != nil {
		if db.IsNotFound(err) {
			return PollResult{}, loginNotFound()
		}
		return PollResult{}, err
	}
	if row.Status == statusConsumed {
		return PollResult{}, alreadyIssued()
	}
	if closed, ok := s.closedPoll(row); ok {
		return closed, nil
	}
	if row.Status != statusApproved || !row.ApproverUserID.Valid {
		return waiting(row.PollIntervalSeconds), nil
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
			return alreadyIssued()
		}
		if locked.Status != statusApproved || !locked.ApproverUserID.Valid {
			result = waiting(locked.PollIntervalSeconds)
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
			result = denied(locked.PollIntervalSeconds, "This workspace cannot approve a CLI login.")
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
			Version: principal.Version,
			Type:    principal.TypeJWT,
			Subject: principal.Subject{
				ID:   locked.ApproverUserID.String,
				Name: locked.ApproverName.String,
				Type: principal.SubjectTypeUser,
			},
			Source:                principal.JWTSource{Header: nil, Payload: nil, Roles: roles, Signature: ""},
			AuthorizedWorkspaceID: locked.WorkspaceID.String,
			Permissions:           workos.PermissionsForRoles(locked.WorkspaceID.String, roles),
		}
		if err := approver.Authorize(rbac.U(urn.New().Workspace(approver.AuthorizedWorkspaceID).RootKey("*"), rbacpermissions.Write)); err != nil {
			if err := db.Query.UpdateCLIDeviceLoginStatus(ctx, tx, db.UpdateCLIDeviceLoginStatusParams{Status: statusDenied, ID: locked.ID}); err != nil {
				return err
			}
			result = denied(locked.PollIntervalSeconds, "You no longer have permission to create this root key.")
			return nil
		}
		validated, err := principalpermissions.ValidateDelegatedPermissions(ctx, approver, requested)
		if err != nil {
			if err := db.Query.UpdateCLIDeviceLoginStatus(ctx, tx, db.UpdateCLIDeviceLoginStatusParams{Status: statusDenied, ID: locked.ID}); err != nil {
				return err
			}
			result = denied(locked.PollIntervalSeconds, fault.UserFacingMessage(err))
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
		n, err := db.Query.ConsumeCLIDeviceLogin(ctx, tx, db.ConsumeCLIDeviceLoginParams{
			KeyID: sql.NullString{String: created.KeyID, Valid: true},
			ID:    locked.ID,
		})
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
			Detail:   "",
			KeyID:    created.KeyID,
			Key:      &secret,
		}
		return nil
	})
	return result, err
}

func (s *Service) closedPoll(row db.CliDeviceLogin) (PollResult, bool) {
	if s.expired(row) && row.Status != statusConsumed && row.Status != statusDenied {
		return expiredPoll(row.PollIntervalSeconds), true
	}
	switch row.Status {
	case statusDenied:
		return denied(row.PollIntervalSeconds, "Authorization was denied."), true
	case statusExpired:
		return expiredPoll(row.PollIntervalSeconds), true
	default:
		return waiting(row.PollIntervalSeconds), false
	}
}

func waiting(interval int32) PollResult {
	return PollResult{Status: PollPermissionsRequired, Interval: interval, Detail: "", KeyID: "", Key: nil}
}

func denied(interval int32, detail string) PollResult {
	return PollResult{Status: PollDenied, Interval: interval, Detail: detail, KeyID: "", Key: nil}
}

func expiredPoll(interval int32) PollResult {
	return PollResult{Status: PollExpired, Interval: interval, Detail: "The login code expired. Run unkey login again.", KeyID: "", Key: nil}
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

func (s *Service) limit(ctx context.Context, action string, remoteIP string, perMinute int64) error {
	if s.Ratelimit == nil {
		return nil
	}
	identifier := remoteIP
	if identifier == "" {
		identifier = "unknown"
	}
	resp, err := s.Ratelimit.Ratelimit(ctx, ratelimit.RatelimitRequest{
		WorkspaceID: ratelimitScope,
		Namespace:   ratelimitScope + "." + action,
		Identifier:  identifier,
		Limit:       perMinute,
		Duration:    time.Minute,
		Cost:        1,
		Time:        time.Time{}, //nolint:exhaustruct // use ratelimiter's clock
	})
	if err != nil {
		// Matches the workspace limiter: an unavailable limiter must not block login.
		logger.Error("cli device login rate limit: ratelimiter error", "action", action, "error", err.Error())
		return nil
	}
	if !resp.Success {
		return fault.New("cli device login rate limit exceeded",
			fault.Code(codes.User.TooManyRequests.IPRateLimited.URN()),
			fault.Public("Too many CLI login requests from this address. Try again in a minute."),
		)
	}
	return nil
}

func (s *Service) deleteExpired(ctx context.Context) {
	cutoff := s.Clock.Now().Add(-retainExpiredFor).UnixMilli()
	if _, err := db.Query.DeleteExpiredCLIDeviceLogins(ctx, s.DB.RW(), cutoff); err != nil {
		logger.Error("cli device login cleanup failed", "error", err.Error())
	}
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

func dashboardSession(p *principal.Principal) (principal.JWTSource, error) {
	if p == nil {
		return principal.JWTSource{}, fault.New("cli login requires a dashboard session",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Public("Sign in to the dashboard to manage this login."),
		)
	}
	source, ok := p.Source.(principal.JWTSource)
	if !ok || len(source.Roles) == 0 {
		return principal.JWTSource{}, fault.New("cli login requires a dashboard session",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Public("Sign in to the dashboard to manage this login."),
		)
	}
	return source, nil
}

func loginView(row db.CliDeviceLogin) LoginView {
	return LoginView{
		UserCode:           row.UserCode,
		DeviceName:         row.DeviceName.String,
		RequesterIP:        row.RequesterIp.String,
		RequesterUserAgent: row.RequesterUserAgent.String,
		Status:             row.Status,
		CreatedAt:          row.CreatedAt,
		ExpiresAt:          row.ExpiresAt,
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

func truncate(value string, maxRunes int) string {
	if runes := []rune(value); len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return value
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

func alreadyIssued() error {
	return invalid("This login already issued a root key. If this CLI did not store it, delete that key in the dashboard and run unkey login again.")
}

func loginNotFound() error {
	return invalid("This login code was not found. Run unkey login again.")
}
