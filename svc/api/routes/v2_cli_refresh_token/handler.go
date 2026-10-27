package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliRefreshTokenRequestBody
type Response = openapi.V2CliRefreshTokenResponseBody

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.refreshToken" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	tokens, err := h.Devices.Refresh(ctx, clidevice.RefreshRequest{
		RefreshToken: req.RefreshToken,
		RemoteIP:     s.Location(),
		UserAgent:    s.UserAgent(),
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2CliRefreshTokenResponseData{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresIn:    tokens.ExpiresIn,
			WorkspaceId:  tokens.WorkspaceID,
			SessionId:    tokens.SessionID,
		},
	})
}
