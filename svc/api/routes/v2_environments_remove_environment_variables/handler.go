package handler

import (
	"context"
	"fmt"
	"net/http"
	"slices"

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

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/environments.removeEnvironmentVariables"
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

	existing, err := db.Query.FindAppEnvVarsByAppAndEnv(ctx, h.DB.RO(), db.FindAppEnvVarsByAppAndEnvParams{
		AppID:         env.AppID,
		EnvironmentID: env.ID,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment variables."),
		)
	}

	keys := make([]string, 0, len(existing))
	for _, e := range existing {
		if slices.Contains(req.Variables, e.Key) {
			keys = append(keys, e.Key)
		}
	}

	if len(keys) > 0 {
		err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			if err := db.Query.DeleteAppEnvVarsByKeys(ctx, tx, db.DeleteAppEnvVarsByKeysParams{
				EnvironmentID: env.ID,
				EnvKeys:       keys,
			}); err != nil {
				return fault.Wrap(
					err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("unable to remove variables"),
					fault.Public("We're unable to remove the environment variables."),
				)
			}

			return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
				{
					WorkspaceID:   principal.WorkspaceID,
					Event:         auditlog.EnvironmentUpdateEvent,
					Display:       fmt.Sprintf("Removed environment variables from environment %s", env.ID),
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
							Meta:        map[string]any{"keys": keys},
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
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
