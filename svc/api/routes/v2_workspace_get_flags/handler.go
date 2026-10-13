package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/flags"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Response = openapi.V2WorkspaceGetFlagsResponseBody

type Handler struct {
	Flags flags.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/workspace.getFlags" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	err = principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Flags(), permissions.Read))
	if err != nil {
		return err
	}

	values, err := h.Flags.Values(ctx, principal.AuthorizedWorkspaceID)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to read the workspace's flags."),
		)
	}

	data := make(map[string]openapi.V2WorkspaceGetFlagsFlag, len(values))
	for slug, flag := range values {
		data[slug] = openapi.V2WorkspaceGetFlagsFlag{
			Enabled:     flag.Value,
			Description: flag.Description,
			AllowOptIn:  flag.AllowEnable,
			AllowOptOut: flag.AllowDisable,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
