package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2EnvironmentsSetEnvironmentVariablesRequestBody
	Response = openapi.V2EnvironmentsSetEnvironmentVariablesResponseBody
)

type Handler struct {
	DB        db.Database
	Vault     vault.VaultServiceClient
	Auditlogs auditlogs.AuditLogService
}

type encryptedValue struct {
	id         string
	ciphertext string
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/environments.setEnvironmentVariables"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	env, err := db.Query.FindEnvironmentByAppAndIdOrSlug(ctx, h.DB.RO(), db.FindEnvironmentByAppAndIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment not found"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.SetEnvironmentVariables,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   env.ID,
			Action:       rbac.SetEnvironmentVariables,
		}),
	))
	if err != nil {
		return err
	}

	byKey := make(map[string]openapi.EnvironmentVariableInput, len(req.Variables))
	keys := make([]string, 0, len(req.Variables))
	for _, v := range req.Variables {
		if _, dup := byKey[v.Key]; dup {
			return fault.New(
				"duplicate variable key",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("duplicate variable key in request"),
				fault.Public(fmt.Sprintf("Variable %q is listed more than once. Each key may appear at most once.", v.Key)),
			)
		}
		byKey[v.Key] = v
		keys = append(keys, v.Key)
	}

	encrypted, err := h.encryptValues(ctx, env.ID, keys, byKey)
	if err != nil {
		return err
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		if _, lockErr := db.Query.LockEnvironmentForUpdate(ctx, tx, env.ID); lockErr != nil {
			if db.IsNotFound(lockErr) {
				return fault.New(
					"environment not found",
					fault.Code(codes.Data.Environment.NotFound.URN()),
					fault.Internal("environment deleted before lock"),
					fault.Public("The requested environment does not exist."),
				)
			}
			return setVarsDBError(lockErr, "unable to lock environment")
		}

		currentKeys, err := db.Query.ListAppEnvVarKeys(ctx, tx, env.ID)
		if err != nil {
			return setVarsDBError(err, "database error")
		}
		existing := make(map[string]struct{}, len(currentKeys))
		for _, key := range currentKeys {
			existing[key] = struct{}{}
		}

		if err := db.Query.DeleteAppEnvVarsByEnvironmentId(ctx, tx, env.ID); err != nil {
			return setVarsDBError(err, "unable to delete variables")
		}

		newEnvVars := buildVariables(keys, byKey, encrypted, env, time.Now().UnixMilli())
		if err = db.BulkQuery.InsertAppEnvironmentVariables(ctx, tx, newEnvVars); err != nil {
			return setVarsDBError(err, "unable to insert variables")
		}

		return h.Auditlogs.Insert(ctx, tx, h.variableAuditLogs(principal, s, env, keys, byKey, existing))
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

func buildVariables(
	keys []string,
	byKey map[string]openapi.EnvironmentVariableInput,
	encrypted map[string]encryptedValue,
	env db.Environment,
	now int64,
) []db.InsertAppEnvironmentVariableParams {
	params := make([]db.InsertAppEnvironmentVariableParams, 0, len(keys))

	for _, key := range keys {
		v := byKey[key]

		varType := db.AppEnvironmentVariablesTypeWriteonly
		if v.Kind != nil && *v.Kind == openapi.Recoverable {
			varType = db.AppEnvironmentVariablesTypeRecoverable
		}

		description := sql.NullString{}
		if v.Description != nil {
			description = sql.NullString{Valid: true, String: *v.Description}
		}

		params = append(params, db.InsertAppEnvironmentVariableParams{
			ID:            encrypted[key].id,
			WorkspaceID:   env.WorkspaceID,
			AppID:         env.AppID,
			EnvironmentID: env.ID,
			EnvKey:        key,
			Value:         encrypted[key].ciphertext,
			Type:          varType,
			Description:   description,
			CreatedAt:     now,
		})
	}

	return params
}

func (h *Handler) variableAuditLogs(
	principal *authprincipal.Principal,
	s *zen.Session,
	env db.Environment,
	keys []string,
	byKey map[string]openapi.EnvironmentVariableInput,
	existing map[string]struct{},
) []auditlog.AuditLog {
	event := func(key, action string) auditlog.AuditLog {
		var display string
		switch action {
		case "created":
			display = fmt.Sprintf("Created environment variable %s in environment %s", key, env.ID)
		case "updated":
			display = fmt.Sprintf("Updated environment variable %s in environment %s", key, env.ID)
		default:
			display = fmt.Sprintf("Removed environment variable %s from environment %s", key, env.ID)
		}

		return auditlog.AuditLog{
			WorkspaceID:   principal.WorkspaceID,
			Event:         auditlog.EnvironmentUpdateEvent,
			Display:       display,
			ActorID:       principal.Subject.ID,
			ActorName:     principal.Subject.Name,
			ActorMeta:     map[string]any{},
			ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					ID:          env.ID,
					Type:        auditlog.EnvironmentResourceType,
					Meta:        map[string]any{"key": key, "action": action},
					Name:        env.Slug,
					DisplayName: env.Slug,
				},
			},
		}
	}

	logs := make([]auditlog.AuditLog, 0, len(keys)+len(existing))
	for _, key := range keys {
		action := "created"
		if _, existed := existing[key]; existed {
			action = "updated"
		}
		logs = append(logs, event(key, action))
	}
	for key := range existing {
		if _, stillSet := byKey[key]; stillSet {
			continue
		}
		logs = append(logs, event(key, "removed"))
	}
	return logs
}

func (h *Handler) encryptValues(
	ctx context.Context,
	environmentID string,
	keys []string,
	byKey map[string]openapi.EnvironmentVariableInput,
) (map[string]encryptedValue, error) {
	out := make(map[string]encryptedValue, len(keys))
	if len(keys) == 0 {
		return out, nil
	}

	if h.Vault == nil {
		return nil, fault.New(
			"vault not configured",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("vault not configured"),
			fault.Public("Environment variables are not available on this deployment."),
		)
	}

	// The vault item key is the variable's id, not its name, so the mapping
	// holds regardless of key contents (mirrors the dashboard env-var flow).
	ids := make(map[string]string, len(keys))
	items := make(map[string]string, len(keys))
	for _, key := range keys {
		id := uid.New(uid.EnvironmentVariablePrefix)
		ids[key] = id
		items[id] = byKey[key].Value
	}

	encrypted, err := h.Vault.EncryptBulk(ctx, &vaultv1.EncryptBulkRequest{
		Keyring: environmentID,
		Items:   items,
	})
	if err != nil {
		return nil, fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("vault error"),
			fault.Public("Failed to encrypt environment variables."),
		)
	}

	for _, key := range keys {
		id := ids[key]
		item, ok := encrypted.GetItems()[id]
		if !ok {
			return nil, fault.New(
				"missing ciphertext",
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("vault returned no ciphertext for id"),
				fault.Public("Failed to encrypt environment variables."),
			)
		}
		out[key] = encryptedValue{id: id, ciphertext: item.GetEncrypted()}
	}

	return out, nil
}

func setVarsDBError(err error, internal string) error {
	return fault.Wrap(
		err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal(internal),
		fault.Public("We're unable to set the environment variables."),
	)
}
