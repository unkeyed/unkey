package handler

import (
	"context"
	"net/http"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2EnvironmentsListEnvironmentVariablesRequestBody
	Response = openapi.V2EnvironmentsListEnvironmentVariablesResponseBody
)

type Handler struct {
	DB    db.Database
	Vault vault.VaultServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/environments.listEnvironmentVariables"
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
			Action:       rbac.ReadEnvironmentVariables,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   env.ID,
			Action:       rbac.ReadEnvironmentVariables,
		}),
	))
	if err != nil {
		return err
	}

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")

	rows, err := db.Query.ListAppEnvVarsByAppAndEnv(ctx, h.DB.RO(), db.ListAppEnvVarsByAppAndEnvParams{
		AppID:         env.AppID,
		EnvironmentID: env.ID,
		IDCursor:      cursor,
		Limit:         int32(limit + 1), // nolint:gosec // limit is bounded [1,100]
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment variables."),
		)
	}

	hasMore := len(rows) > limit
	var nextCursor *string
	if hasMore {
		nextCursor = ptr.P(rows[limit].ID)
		rows = rows[:limit]
	}

	// Bulk-decrypt every recoverable variable in a single vault round-trip. The
	// keyring is the environment id, matching how the set handler encrypts them.
	// Writeonly variables are never decrypted and never expose a value.
	bulkItems := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Type == db.AppEnvironmentVariablesTypeRecoverable {
			bulkItems[r.ID] = r.Value
		}
	}

	plaintext := map[string]string{}
	if len(bulkItems) > 0 {
		if h.Vault == nil {
			return fault.New(
				"vault not configured",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("vault not configured"),
				fault.Public("Environment variables are not available on this deployment."),
			)
		}

		decrypted, derr := h.Vault.DecryptBulk(ctx, &vaultv1.DecryptBulkRequest{
			Keyring: env.ID,
			Items:   bulkItems,
		})
		if derr != nil {
			return fault.Wrap(
				derr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("vault error"),
				fault.Public("Failed to decrypt environment variables."),
			)
		}
		plaintext = decrypted.GetItems()

		for id := range bulkItems {
			if _, ok := plaintext[id]; !ok {
				return fault.New(
					"missing plaintext",
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("vault returned no plaintext for id"),
					fault.Public("Failed to decrypt environment variables."),
				)
			}
		}
	}

	data := make([]openapi.EnvironmentVariable, len(rows))
	for i, r := range rows {
		item := openapi.EnvironmentVariable{
			Key:         r.Key,
			Kind:        openapi.Writeonly,
			Value:       "",
			Description: r.Description.String,
			CreatedAt:   r.CreatedAt,
		}
		if r.Type == db.AppEnvironmentVariablesTypeRecoverable {
			item.Kind = openapi.Recoverable
			item.Value = plaintext[r.ID]
		}
		data[i] = item
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: &openapi.Pagination{Cursor: nextCursor, HasMore: hasMore},
	})
}
