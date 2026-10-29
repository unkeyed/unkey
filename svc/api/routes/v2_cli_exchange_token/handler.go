package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/grant"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2CliExchangeTokenRequestBody
type Response = openapi.V2CliExchangeTokenResponseBody

type Handler struct {
	Grants *grant.Service
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/cli.exchangeToken" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if req.RefreshToken != nil {
		tokens, err := h.Grants.Refresh(ctx, grant.RefreshRequest{
			RefreshToken: *req.RefreshToken,
			RemoteIP:     s.Location(),
			UserAgent:    s.UserAgent(),
		})
		if err != nil {
			return err
		}
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Data: openapi.V2CliExchangeTokenResponseData{
				AccessToken:  tokens.AccessToken,
				RefreshToken: tokens.RefreshToken,
				ExpiresIn:    tokens.ExpiresIn,
				WorkspaceId:  tokens.WorkspaceID,
				SessionId:    tokens.SessionID,
			},
		})
	}

	result, err := h.Grants.Poll(ctx, grant.PollRequest{
		LoginID:   *req.LoginId,
		RemoteIP:  s.Location(),
		UserAgent: s.UserAgent(),
	})
	if err != nil {
		return err
	}

	switch result.Status {
	case grant.PollPermissionsRequired:
		s.AddHeader("Retry-After", strconv.Itoa(int(result.Interval)))
		return s.JSON(http.StatusAccepted, openapi.V2CliExchangeTokenPendingResponseBody{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Data: openapi.EmptyResponse{},
		})
	case grant.PollDenied:
		return fault.New("device login denied",
			fault.Code(codes.Auth.Authorization.Forbidden.URN()),
			fault.Public(result.Detail),
		)
	case grant.PollExpired:
		return fault.New("device login expired",
			fault.Code(codes.Auth.Authentication.DeviceLoginExpired.URN()),
			fault.Public(result.Detail),
		)
	case grant.PollComplete:
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Data: openapi.V2CliExchangeTokenResponseData{
				AccessToken:  result.AccessToken,
				RefreshToken: result.RefreshToken,
				ExpiresIn:    result.ExpiresIn,
				WorkspaceId:  result.WorkspaceID,
				SessionId:    result.SessionID,
			},
		})
	default:
		return fault.New("unexpected device login state",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
		)
	}
}
