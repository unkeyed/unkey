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
	Request  = openapi.V2EnvironmentsAddEnvironmentVariablesRequestBody
	Response = openapi.V2EnvironmentsAddEnvironmentVariablesResponseBody
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

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/environments.addEnvironmentVariables"
}

// Handle processes the HTTP request
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

	deduped := make(map[string]openapi.EnvironmentVariableInput, len(req.Variables))
	keys := make([]string, 0, len(req.Variables))
	for _, v := range req.Variables {
		if _, ok := deduped[v.Key]; !ok {
			keys = append(keys, v.Key)
		}
		deduped[v.Key] = v
	}

	encrypted, err := h.encryptValues(ctx, env.ID, keys, deduped)
	if err != nil {
		return err
	}

	var currentVars []db.ListAppEnvVarsForSetRow
	var newParams []db.InsertAppEnvironmentVariableParams

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
			return addVarsDBError(lockErr, "unable to lock environment")
		}

		rows, listErr := db.Query.ListAppEnvVarsForSet(ctx, tx, env.ID)
		if listErr != nil {
			return addVarsDBError(listErr, "database error")
		}
		currentVars = rows
		currentByKey := make(map[string]db.ListAppEnvVarsForSetRow, len(currentVars))
		for _, v := range currentVars {
			currentByKey[v.Key] = v
		}

		now := time.Now().UnixMilli()
		newParams = make([]db.InsertAppEnvironmentVariableParams, 0, len(keys))
		auditLogs := make([]auditlog.AuditLog, 0, len(keys))

		for _, key := range keys {
			if _, existed := currentByKey[key]; existed {
				continue
			}
			v := deduped[key]

			varType := db.AppEnvironmentVariablesTypeRecoverable
			if ptr.SafeDeref(v.Sensitive, false) {
				varType = db.AppEnvironmentVariablesTypeWriteonly
			}

			description := sql.NullString{}
			if v.Description != nil {
				description = sql.NullString{Valid: true, String: *v.Description}
			}

			deleteProtection := ptr.SafeDeref(v.DeleteProtection, false)

			newParams = append(newParams, db.InsertAppEnvironmentVariableParams{
				ID:               encrypted[key].id,
				WorkspaceID:      env.WorkspaceID,
				AppID:            env.AppID,
				EnvironmentID:    env.ID,
				EnvKey:           key,
				Value:            encrypted[key].ciphertext,
				Type:             varType,
				Description:      description,
				DeleteProtection: sql.NullBool{Valid: true, Bool: deleteProtection},
				CreatedAt:        now,
			})

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.EnvironmentUpdateEvent,
				Display:       fmt.Sprintf("Created environment variable %s in environment %s", key, env.ID),
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
						Meta:        map[string]any{"key": key, "action": "created"},
						Name:        env.Slug,
						DisplayName: env.Slug,
					},
				},
			})
		}

		if len(newParams) == 0 {
			return nil
		}

		if insErr := db.BulkQuery.InsertAppEnvironmentVariables(ctx, tx, newParams); insErr != nil {
			return addVarsDBError(insErr, "unable to insert variables")
		}

		return h.Auditlogs.Insert(ctx, tx, auditLogs)
	})
	if err != nil {
		return err
	}

	data := make([]openapi.EnvironmentVariableMetadata, 0, len(currentVars)+len(newParams))
	for _, cur := range currentVars {
		var desc *string
		if cur.Description.Valid {
			d := cur.Description.String
			desc = &d
		}
		data = append(data, openapi.EnvironmentVariableMetadata{
			Key:              cur.Key,
			Sensitive:        cur.Type == db.AppEnvironmentVariablesTypeWriteonly,
			Description:      desc,
			DeleteProtection: cur.DeleteProtection.Valid && cur.DeleteProtection.Bool,
		})
	}
	for _, p := range newParams {
		var desc *string
		if p.Description.Valid {
			d := p.Description.String
			desc = &d
		}
		data = append(data, openapi.EnvironmentVariableMetadata{
			Key:              p.EnvKey,
			Sensitive:        p.Type == db.AppEnvironmentVariablesTypeWriteonly,
			Description:      desc,
			DeleteProtection: p.DeleteProtection.Valid && p.DeleteProtection.Bool,
		})
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}

func (h *Handler) encryptValues(
	ctx context.Context,
	environmentID string,
	keys []string,
	deduped map[string]openapi.EnvironmentVariableInput,
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

	ids := make(map[string]string, len(keys))
	items := make(map[string]string, len(keys))
	for _, key := range keys {
		id := uid.New(uid.EnvironmentVariablePrefix)
		ids[key] = id
		items[id] = deduped[key].Value
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

func addVarsDBError(err error, internal string) error {
	return fault.Wrap(
		err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal(internal),
		fault.Public("We're unable to add the environment variables."),
	)
}
