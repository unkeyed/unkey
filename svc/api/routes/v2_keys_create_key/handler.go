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
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2KeysCreateKeyRequestBody
	Response = openapi.V2KeysCreateKeyResponseBody
)

// Handler creates keys through the v2 API route.
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
	return "/v2/keys.createKey"
}

// Handle resolves the API and creates its key.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	api, err := db.Query.FindApiByID(ctx, h.DB.RO(), req.ApiId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("api not found",
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("api not found"),
				fault.Public("The specified API was not found."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve API."),
		)
	}

	if api.WorkspaceID != principal.WorkspaceID {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api belongs to different workspace"),
			fault.Public("The specified API was not found."),
		)
	}

	// The key does not exist yet, so authorize creation against its keyspace.
	// Keep the API checks while v2 clients still use API-scoped permissions.
	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   req.ApiId,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Keyspace(api.KeyAuthID.String),
			permissions.CreateKey{},
		),
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Api.NotFound.URN(),
			"The specified API was not found.",
		)
	}

	keyspace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), api.KeyAuthID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("api not set up for keys",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("api not set up for keys, keyspace not found"),
				fault.Public("The requested API is not set up to handle keys."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve API information."),
		)
	}

	return h.create(ctx, s, principal, req, api, keyspace)
}
