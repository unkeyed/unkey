package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2EnvironmentsRemoveEnvironmentVariablesRequestBody
	Response = openapi.V2EnvironmentsRemoveEnvironmentVariablesResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/environments.removeEnvironmentVariables"
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

	seen := make(map[string]struct{}, len(req.Variables))
	keys := make([]string, 0, len(req.Variables))
	for _, key := range req.Variables {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}

	currentVars, err := db.Query.ListAppEnvVarsForSet(ctx, h.DB.RO(), env.ID)
	if err != nil {
		return removeVarsDBError(err, "database error")
	}
	currentByKey := make(map[string]db.ListAppEnvVarsForSetRow, len(currentVars))
	for _, v := range currentVars {
		currentByKey[v.Key] = v
	}

	removed := make(map[string]struct{}, len(keys))
	toRemove := make([]string, 0, len(keys))
	auditLogs := make([]auditlog.AuditLog, 0, len(keys))
	for _, key := range keys {
		cur, ok := currentByKey[key]
		if !ok {
			continue
		}
		if cur.DeleteProtection.Valid && cur.DeleteProtection.Bool {
			continue
		}
		removed[key] = struct{}{}
		toRemove = append(toRemove, key)
		auditLogs = append(auditLogs, auditlog.AuditLog{
			WorkspaceID:   principal.WorkspaceID,
			Event:         auditlog.EnvironmentUpdateEvent,
			Display:       fmt.Sprintf("Removed environment variable %s from environment %s", key, env.ID),
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
					Meta:        map[string]any{"key": key, "action": "removed"},
					Name:        env.Slug,
					DisplayName: env.Slug,
				},
			},
		})
	}

	if len(toRemove) > 0 {
		err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			if delErr := db.Query.DeleteUnprotectedAppEnvVarsByKeys(ctx, tx, db.DeleteUnprotectedAppEnvVarsByKeysParams{
				EnvironmentID: env.ID,
				EnvKeys:       toRemove,
			}); delErr != nil {
				return removeVarsDBError(delErr, "unable to remove variables")
			}
			return h.Auditlogs.Insert(ctx, tx, auditLogs)
		})
		if err != nil {
			return err
		}
	}

	data := make([]openapi.EnvironmentVariableMetadata, 0, len(currentVars))
	for _, cur := range currentVars {
		if _, gone := removed[cur.Key]; gone {
			continue
		}
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

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}

func removeVarsDBError(err error, internal string) error {
	return fault.Wrap(
		err,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal(internal),
		fault.Public("We're unable to remove the environment variables."),
	)
}
