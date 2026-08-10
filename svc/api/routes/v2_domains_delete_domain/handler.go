package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/customdomain"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsDeleteDomainRequestBody
	Response = openapi.V2DomainsDeleteDomainResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.CustomDomainServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/domains.deleteDomain"
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

	identifier := domaingate.CanonicalizeIdentifier(req.Domain)

	row, err := db.Query.FindCustomDomainByIdentifier(ctx, h.DB.RO(), db.FindCustomDomainByIdentifierParams{
		WorkspaceID: principal.WorkspaceID,
		Domain:      identifier,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Domain.NotFound.URN()),
				fault.Internal("domain not found"),
				fault.Public("The requested domain does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve domain."),
		)
	}

	if err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.DeleteDomain,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   row.EnvironmentID,
			Action:       rbac.DeleteDomain,
		}),
	)); err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Domain.NotFound.URN(),
			"The requested domain does not exist.",
		)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = h.CtrlClient.DeleteCustomDomain(ctx, &ctrlv1.DeleteCustomDomainRequest{
		WorkspaceId: principal.WorkspaceID,
		ProjectId:   row.ProjectID,
		Domain:      row.Domain,
		Actor:       actor,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Domain.NotFound.URN()),
				fault.Internal("domain not found"),
				fault.Public("The requested domain does not exist."),
			)
		}
		return customdomain.MapCtrlError(err, "delete custom domain")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
