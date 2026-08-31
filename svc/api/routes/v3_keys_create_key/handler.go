package handler

import (
	"context"

	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V3KeysCreateKeyRequestBody
	Response = openapi.V2KeysCreateKeyResponseBody
)

// Handler creates keys through the v3 API route.
type Handler struct {
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Vault     vault.VaultServiceClient
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string {
	return "/v3/keys.createKey"
}

// Handle resolves the keyspace and creates its key.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	keyspace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), req.Keyspace)
	if err != nil {
		if db.IsNotFound(err) {
			return keyspaceNotFound("keyspace not found")
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve the keyspace."),
		)
	}

	if keyspace.WorkspaceID != principal.WorkspaceID {
		return keyspaceNotFound("keyspace belongs to a different workspace")
	}
	if keyspace.DeletedAtM.Valid {
		return keyspaceNotFound("keyspace is deleted")
	}

	liveAPIs, err := db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: principal.WorkspaceID,
		KeyAuthIds:  []string{keyspace.ID},
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve the keyspace."),
		)
	}
	if len(liveAPIs) == 0 {
		return keyspaceNotFound("keyspace does not belong to a live API")
	}
	apiID := liveAPIs[0].ApiID

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   apiID,
			Action:       rbac.CreateKey,
		}),
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.KeyAuth.NotFound.URN(),
			"The specified keyspace was not found.",
		)
	}

	return h.create(ctx, s, principal, req, apiID, keyspace)
}

// keyspaceNotFound hides whether a keyspace does not exist or belongs to a different workspace.
func keyspaceNotFound(internal string) error {
	return fault.New("keyspace not found",
		fault.Code(codes.Data.KeyAuth.NotFound.URN()),
		fault.Internal(internal),
		fault.Public("The specified keyspace was not found."),
	)
}
