package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/domain/domaingate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/domain"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DomainsGetDomainRequestBody
	Response = openapi.V2DomainsGetDomainResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/domains.getDomain"
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
			return fault.New(
				"domain not found",
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
			Action:       rbac.ReadDomain,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   row.EnvironmentID,
			Action:       rbac.ReadDomain,
		}),
	)); err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Domain.NotFound.URN(),
			"The requested domain does not exist.",
		)
	}

	data := openapi.Domain{
		Id:                row.ID,
		Domain:            row.Domain,
		ProjectId:         row.ProjectID,
		AppId:             row.AppID,
		EnvironmentId:     row.EnvironmentID,
		Status:            domain.Status(row.VerificationStatus),
		VerificationError: nil,
		DnsRecords: domain.DnsRecords(domain.DnsRecordsInput{
			Domain:            row.Domain,
			TargetCname:       row.TargetCname,
			VerificationToken: row.VerificationToken,
			RoutingVerified:   row.CnameVerified,
			OwnershipVerified: row.OwnershipVerified,
		}),
		CreatedAt: row.CreatedAt,
		UpdatedAt: nil,
	}
	if row.VerificationError.Valid && row.VerificationError.String != "" {
		data.VerificationError = ptr.P(row.VerificationError.String)
	}
	if row.UpdatedAt.Valid {
		data.UpdatedAt = ptr.P(row.UpdatedAt.Int64)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
