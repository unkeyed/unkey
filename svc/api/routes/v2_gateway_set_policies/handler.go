package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/policyconfig"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/encoding/protojson"
)

type (
	Request  = openapi.V2GatewaySetPoliciesRequestBody
	Response = openapi.V2GatewaySetPoliciesResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/gateway.setPolicies"
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

	env, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
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

	policyURN := urn.New().Workspace(principal.WorkspaceID).Project(env.ProjectID).App(env.AppID).Environment(env.ID).Gateway().Policy("*")
	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.SetPolicies,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   env.ID,
			Action:       rbac.SetPolicies,
		}),
		rbac.And(
			rbac.U(policyURN, permissions.WriteGatewayPolicy{}),
			rbac.U(policyURN, permissions.DeleteGatewayPolicy{}),
		),
	))
	if err != nil {
		return err
	}

	policies, err := policyconfig.ToProto(req.Policies)
	if err != nil {
		return err
	}

	var keyspaceIDs []string
	for _, p := range policies {
		keyspaceIDs = append(keyspaceIDs, p.GetKeyauth().GetKeySpaceIds()...)
	}
	if len(keyspaceIDs) > 0 {
		found, err := db.Query.FindKeyAuthsByIdsAndWorkspace(ctx, h.DB.RO(), db.FindKeyAuthsByIdsAndWorkspaceParams{
			WorkspaceID: principal.WorkspaceID,
			KeyAuthIds:  keyspaceIDs,
		})
		if err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to verify keyspaces"),
				fault.Public("We're unable to set the policies."),
			)
		}
		for _, id := range keyspaceIDs {
			if !slices.ContainsFunc(found, func(row db.FindKeyAuthsByIdsAndWorkspaceRow) bool {
				return row.ID == id && row.ProjectID == env.ProjectID
			}) {
				return fault.New(
					"keyspace not found",
					fault.Code(codes.Data.KeySpace.NotFound.URN()),
					fault.Internal("keyspace not found in project"),
					fault.Public(fmt.Sprintf("Keyspace %q does not exist.", id)),
				)
			}
		}
	}

	newLog := func(display string, meta map[string]any) auditlog.AuditLog {
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
					Meta:        meta,
					Name:        env.Slug,
					DisplayName: env.Slug,
				},
			},
		}
	}

	auditLogs := make([]auditlog.AuditLog, 0, len(policies)+1)
	for _, p := range policies {
		// The exact document as stored, so the audit trail alone
		// reconstructs every configuration the environment had.
		doc, marshalErr := protojson.Marshal(p)
		if marshalErr != nil {
			return fault.Wrap(
				marshalErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("policy serialization failed"),
				fault.Public("We're unable to set the policies."),
			)
		}
		auditLogs = append(auditLogs, newLog(
			fmt.Sprintf("Set policy %s (%s) for environment %s", p.GetName(), p.GetId(), env.ID),
			map[string]any{
				"policyId":   p.GetId(),
				"policyType": policyconfig.VariantName(p),
				"policy":     json.RawMessage(doc),
			},
		))
	}

	// An empty request wipes everything but yields no per-policy logs,
	// so record the destructive action on its own.
	if len(auditLogs) == 0 {
		auditLogs = append(auditLogs, newLog(
			fmt.Sprintf("Removed all policies for environment %s", env.ID),
			map[string]any{},
		))
	}

	blob, err := policyconfig.Marshal(policies)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("policy serialization failed"),
			fault.Public("We're unable to set the policies."),
		)
	}

	now := time.Now().UnixMilli()
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		if upsertErr := db.Query.UpsertAppRuntimeSettingsPolicyConfig(ctx, tx, db.UpsertAppRuntimeSettingsPolicyConfigParams{
			WorkspaceID:    env.WorkspaceID,
			AppID:          env.AppID,
			EnvironmentID:  env.ID,
			SentinelConfig: blob,
			CreatedAt:      now,
			UpdatedAt:      sql.NullInt64{Valid: true, Int64: now},
		}); upsertErr != nil {
			return fault.Wrap(
				upsertErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to write gateway policy config"),
				fault.Public("We're unable to set the policies."),
			)
		}

		return h.Auditlogs.Insert(ctx, tx, auditLogs)
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
