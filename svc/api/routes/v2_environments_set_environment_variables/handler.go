package handler

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
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
	}

	encrypted, err := h.encryptValues(ctx, env.ID, byKey)
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	newEnvVars := make([]db.InsertAppEnvironmentVariableParams, 0, len(byKey))
	for key, v := range byKey {
		varType := db.AppEnvironmentVariablesTypeWriteonly
		if ptr.SafeDeref(v.Kind, openapi.Writeonly) == openapi.Recoverable {
			varType = db.AppEnvironmentVariablesTypeRecoverable
		}

		description := sql.NullString{}
		if v.Description != nil {
			description = sql.NullString{Valid: true, String: *v.Description}
		}

		newEnvVars = append(newEnvVars, db.InsertAppEnvironmentVariableParams{
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
			return fault.Wrap(
				lockErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to lock environment"),
				fault.Public("We're unable to set the environment variables."),
			)
		}

		if err := db.Query.DeleteAppEnvVarsByEnvironmentId(ctx, tx, env.ID); err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to delete variables"),
				fault.Public("We're unable to set the environment variables."),
			)
		}

		if err = db.BulkQuery.InsertAppEnvironmentVariables(ctx, tx, newEnvVars); err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to insert variables"),
				fault.Public("We're unable to set the environment variables."),
			)
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.EnvironmentUpdateEvent,
				Display:       fmt.Sprintf("Set environment variables for environment %s", env.ID),
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
						Meta:        map[string]any{"keys": slices.Sorted(maps.Keys(byKey))},
						Name:        env.Slug,
						DisplayName: env.Slug,
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

func (h *Handler) encryptValues(
	ctx context.Context,
	environmentID string,
	byKey map[string]openapi.EnvironmentVariableInput,
) (map[string]encryptedValue, error) {
	out := make(map[string]encryptedValue, len(byKey))
	if len(byKey) == 0 {
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
	ids := make(map[string]string, len(byKey))
	items := make(map[string]string, len(byKey))
	for key, v := range byKey {
		id := uid.New(uid.EnvironmentVariablePrefix)
		ids[key] = id
		items[id] = v.Value
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

	for key, id := range ids {
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
