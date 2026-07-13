package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	frontlinev1 "github.com/unkeyed/unkey/gen/proto/frontline/v1"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/policyconfig"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"google.golang.org/protobuf/encoding/protojson"
)

type (
	Request  = openapi.V2GatewayUpdatePolicyRequestBody
	Response = openapi.V2GatewayUpdatePolicyResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/gateway.updatePolicy"
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

	rules := 0
	for _, set := range []bool{req.Keyauth != nil, req.Ratelimit != nil, req.Firewall != nil, req.Openapi != nil} {
		if set {
			rules++
		}
	}
	if rules > 1 {
		return fault.New(
			"multiple rule variants",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("more than one rule variant in update"),
			fault.Public("At most one of keyauth, ratelimit, firewall or openapi may be set."),
		)
	}
	if rules == 0 && req.Name == nil && req.Enabled == nil && !req.Match.IsSpecified() {
		return fault.New(
			"empty update",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("no updatable fields in request"),
			fault.Public("Provide at least one field to update."),
		)
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
	))
	if err != nil {
		return err
	}

	// Read-modify-write on the config blob: the environment lock serializes
	// this against concurrent updatePolicy calls so neither overwrites the
	// other's change. setPolicies doesn't lock because it never reads first.
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		if _, lockErr := db.Query.LockEnvironmentForUpdate(ctx, tx, env.ID); lockErr != nil {
			if db.IsNotFound(lockErr) {
				// Deleted between the read above and acquiring the lock.
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
				fault.Public("We're unable to update the policy."),
			)
		}

		settings, findErr := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, tx, db.FindAppRuntimeSettingsByAppAndEnvParams{
			AppID:         env.AppID,
			EnvironmentID: env.ID,
		})
		if findErr != nil && !db.IsNotFound(findErr) {
			return fault.Wrap(
				findErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("unable to read sentinel config"),
				fault.Public("We're unable to update the policy."),
			)
		}
		cfg, parseErr := policyconfig.Parse(settings.AppRuntimeSetting.SentinelConfig)
		if parseErr != nil {
			return fault.Wrap(
				parseErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("stored sentinel config is not valid protojson"),
				fault.Public("We're unable to update the policy."),
			)
		}

		// A missing settings row and an empty config both mean the id can't
		// exist, so they collapse into the same not-found below.
		policies := cfg.GetPolicies()
		idx := slices.IndexFunc(policies, func(p *frontlinev1.Policy) bool { return p.GetId() == req.PolicyId })
		if idx < 0 {
			return fault.New(
				"policy not found",
				fault.Code(codes.Data.Policy.NotFound.URN()),
				fault.Internal("policy id not present in stored policies"),
				fault.Public("The requested policy does not exist. Note that policy ids change when the policy list is replaced."),
			)
		}

		// Patch at the API level and reconvert: the openapi->proto pass is
		// the validation pass, so the merged policy is checked exactly like
		// a policy arriving via setPolicies.
		existing, mapErr := policyconfig.PolicyFromProto(policies[idx])
		if mapErr != nil {
			return mapErr
		}
		patched := openapi.Policy{
			Name:      existing.Name,
			Enabled:   existing.Enabled,
			Match:     existing.Match,
			Keyauth:   existing.Keyauth,
			Ratelimit: existing.Ratelimit,
			Firewall:  existing.Firewall,
			Openapi:   existing.Openapi,
		}
		if req.Name != nil {
			patched.Name = *req.Name
		}
		if req.Enabled != nil {
			patched.Enabled = *req.Enabled
		}
		if req.Match.IsSpecified() {
			patched.Match = nil
			if !req.Match.IsNull() {
				match := req.Match.MustGet()
				patched.Match = &match
			}
		}
		if rules == 1 {
			patched.Keyauth = req.Keyauth
			patched.Ratelimit = req.Ratelimit
			patched.Firewall = req.Firewall
			patched.Openapi = req.Openapi
		}

		updated, convErr := policyconfig.PolicyToProto("policy", patched)
		if convErr != nil {
			return convErr
		}
		updated.Id = req.PolicyId
		policies[idx] = updated

		// A keyauth kept from storage was verified when it was written; only
		// a keyauth arriving with this request needs its keyspaces checked.
		// After validation, matching setPolicies' validate-then-verify order.
		if req.Keyauth != nil && len(req.Keyauth.Keyspaces) > 0 {
			found, keyspaceErr := db.Query.FindKeyAuthsByIdsAndWorkspace(ctx, tx, db.FindKeyAuthsByIdsAndWorkspaceParams{
				WorkspaceID: principal.WorkspaceID,
				KeyAuthIds:  req.Keyauth.Keyspaces,
			})
			if keyspaceErr != nil {
				return fault.Wrap(
					keyspaceErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("unable to verify keyspaces"),
					fault.Public("We're unable to update the policy."),
				)
			}
			for _, id := range req.Keyauth.Keyspaces {
				if !slices.Contains(found, id) {
					return fault.New(
						"keyspace not found",
						fault.Code(codes.Data.KeySpace.NotFound.URN()),
						fault.Internal("keyspace not found in workspace"),
						fault.Public(fmt.Sprintf("Keyspace %q does not exist.", id)),
					)
				}
			}
		}

		blob, marshalErr := protojson.Marshal(&frontlinev1.Config{Policies: policies})
		if marshalErr != nil {
			return fault.Wrap(
				marshalErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("policy serialization failed"),
				fault.Public("We're unable to update the policy."),
			)
		}

		now := time.Now().UnixMilli()
		if upsertErr := db.Query.UpsertAppRuntimeSettingsSentinelConfig(ctx, tx, db.UpsertAppRuntimeSettingsSentinelConfigParams{
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
				fault.Internal("unable to write sentinel config"),
				fault.Public("We're unable to update the policy."),
			)
		}

		// The exact document as stored, so the audit trail alone
		// reconstructs every configuration the policy had.
		doc, docErr := protojson.Marshal(updated)
		if docErr != nil {
			return fault.Wrap(
				docErr,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("policy serialization failed"),
				fault.Public("We're unable to update the policy."),
			)
		}
		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   principal.WorkspaceID,
			Event:         auditlog.EnvironmentUpdateEvent,
			Display:       fmt.Sprintf("Updated policy %s (%s) for environment %s", updated.GetName(), updated.GetId(), env.ID),
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
					Meta:        map[string]any{"policyId": updated.GetId(), "policyType": policyconfig.VariantName(updated), "policy": json.RawMessage(doc)},
					Name:        env.Slug,
					DisplayName: env.Slug,
				},
			},
		}})
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
