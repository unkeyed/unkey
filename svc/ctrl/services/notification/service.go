package notification

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auditlogs"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
)

type Config struct {
	Database  db.Database
	Vault     vault.VaultServiceClient
	Directory workos.Directory
	Auditlogs auditlogs.AuditLogService
	Bearer    string
	Clock     clock.Clock
	Restate   *ingress.Client
}

type Service struct {
	ctrlv1connect.UnimplementedNotificationServiceHandler
	db        db.Database
	vault     vault.VaultServiceClient
	directory workos.Directory
	auditlogs auditlogs.AuditLogService
	bearer    string
	clock     clock.Clock
	restate   *ingress.Client
}

func New(cfg Config) *Service {
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}
	return &Service{
		UnimplementedNotificationServiceHandler: ctrlv1connect.UnimplementedNotificationServiceHandler{},
		db:                                      cfg.Database,
		vault:                                   cfg.Vault,
		directory:                               cfg.Directory,
		auditlogs:                               cfg.Auditlogs,
		bearer:                                  cfg.Bearer,
		clock:                                   cfg.Clock,
		restate:                                 cfg.Restate,
	}
}

func (s *Service) authorize(ctx context.Context, req auth.Request, workspaceID string) error {
	if s.bearer == "" {
		return rpcError(connect.CodeUnauthenticated, "notification authentication is not configured")
	}
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return err
	}
	if !validID(workspaceID, 48) {
		return rpcError(connect.CodeInvalidArgument, "workspace_id is required")
	}
	return validateScope(ctx, s.db, workspaceID, "", "", "")
}

func (s *Service) mutate(ctx context.Context, workspaceID string, caller *ctrlv1.ActorInfo, fn func(context.Context, db.DBTX, *db.Queries) error) error {
	if caller == nil {
		return rpcError(connect.CodeInvalidArgument, "actor is required")
	}
	err := db.TxRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		if _, err := queries.LockNotificationWorkspace(ctx, workspaceID); err != nil {
			return err
		}
		return fn(ctx, tx, queries)
	})
	return operationError(err)
}

func (s *Service) audit(ctx context.Context, tx db.DBTX, workspaceID string, caller *ctrlv1.ActorInfo, resource, action, id string) error {
	return s.auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
		WorkspaceID:   workspaceID,
		Event:         auditlog.AuditLogEvent("notification." + resource + "." + action),
		Display:       "Notification " + resource + " " + action,
		ActorID:       caller.GetId(),
		ActorName:     caller.GetName(),
		ActorType:     actor.AuditType(caller.GetType()),
		ActorMeta:     actor.Meta(caller.GetMeta()),
		RemoteIP:      caller.GetRemoteIp(),
		UserAgent:     caller.GetUserAgent(),
		CorrelationID: "",
		Targets:       []auditlog.AuditLogTarget{{ID: id, Type: auditlog.AuditLogTargetType("notification_" + resource), Name: id, DisplayName: id, Meta: nil}},
	}})
}

func validateScope(ctx context.Context, queries db.Querier, workspaceID, projectID, appID, environmentID string) error {
	if (appID != "" && projectID == "") || (environmentID != "" && appID == "") || len(projectID) > 48 || len(appID) > 48 || len(environmentID) > 48 {
		return rpcError(connect.CodeInvalidArgument, "scope must include its parent resources")
	}
	valid, err := queries.NotificationManagementScopeValid(ctx, db.NotificationManagementScopeValidParams{
		WorkspaceID:   workspaceID,
		ProjectID:     projectID,
		AppID:         appID,
		EnvironmentID: environmentID,
	})
	if err != nil {
		return operationError(err)
	}
	if !valid {
		return rpcError(connect.CodeNotFound, "workspace or resource not found")
	}
	return nil
}

func eventTypes() []string {
	definitions := deployment.Definitions()
	types := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		types = append(types, definition.Type())
	}
	return types
}

func validateType(eventType string) error {
	for _, supported := range eventTypes() {
		if eventType == supported {
			return nil
		}
	}
	return rpcError(connect.CodeInvalidArgument, "unsupported notification_type")
}

func stableID(workspaceID, action, key string) string {
	hash := sha256.Sum256([]byte(workspaceID + "\x00" + action + "\x00" + key))
	return "ntf_" + base64.RawURLEncoding.EncodeToString(hash[:])
}

func validID(value string, lengthMax int) bool {
	return value != "" && strings.TrimSpace(value) == value && len(value) <= lengthMax && !strings.ContainsRune(value, '\x00') && utf8.ValidString(value)
}

func validateName(name string) error {
	if strings.TrimSpace(name) == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 128 {
		return rpcError(connect.CodeInvalidArgument, "name must contain between 1 and 128 characters")
	}
	return nil
}

func pageLimit(limit int32, cursor string) (int32, error) {
	if limit < 0 || limit > 100 || len(cursor) > 256 || strings.ContainsRune(cursor, '\x00') {
		return 0, rpcError(connect.CodeInvalidArgument, "invalid pagination")
	}
	if limit == 0 {
		limit = 50
	}
	return limit, nil
}

func nullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

func rpcError(code connect.Code, message string) error {
	return connect.NewError(code, errors.New(message))
}

func operationError(err error) error {
	if err == nil {
		return nil
	}
	var rpc *connect.Error
	if errors.As(err, &rpc) {
		return err
	}
	if db.IsNotFound(err) {
		return rpcError(connect.CodeNotFound, "notification resource not found")
	}
	return rpcError(connect.CodeInternal, "notification operation failed")
}
