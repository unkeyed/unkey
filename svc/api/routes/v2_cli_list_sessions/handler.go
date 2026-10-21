package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliListSessionsRequestBody
type Response = openapi.V2CliListSessionsResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.listSessions" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	if err := clidevice.RequireUserSession(p); err != nil {
		return err
	}

	if _, err := zen.BindBody[Request](s); err != nil {
		return err
	}

	rows, err := h.Devices.List(ctx, clidevice.ListRequest{Principal: p})
	if err != nil {
		return err
	}

	sessions := make([]openapi.V2CliListSessionsSession, 0, len(rows))
	for _, row := range rows {
		permissions := row.Permissions
		if permissions == nil {
			permissions = []string{}
		}
		item := openapi.V2CliListSessionsSession{
			Id:          row.ID,
			UserId:      row.UserID,
			Permissions: permissions,
			CreatedAt:   row.CreatedAt,
			LastUsedAt:  row.LastUsedAt,
			ExpiresAt:   row.ExpiresAt,
		}
		if row.DeviceName != "" {
			item.DeviceName = &row.DeviceName
		}
		if row.RevokedAt != 0 {
			revokedAt := row.RevokedAt
			item.RevokedAt = &revokedAt
		}
		sessions = append(sessions, item)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2CliListSessionsResponseData{Sessions: sessions},
	})
}
