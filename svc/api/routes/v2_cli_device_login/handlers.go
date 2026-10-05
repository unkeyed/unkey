package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Handler struct {
	Devices *clidevice.Service
}

func (h *Handler) start(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[openapi.V2CliStartDeviceLoginRequestBody](s)
	if err != nil {
		return err
	}
	deviceName := ""
	if req.DeviceName != nil {
		deviceName = *req.DeviceName
	}
	started, err := h.Devices.Start(ctx, deviceName)
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, openapi.V2CliStartDeviceLoginResponseBody{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2CliStartDeviceLoginResponseData{
			LoginId:                 started.LoginID,
			UserCode:                started.UserCode,
			VerificationUri:         started.VerificationURI,
			VerificationUriComplete: started.VerificationURIComplete,
			ExpiresIn:               int64(started.ExpiresIn),
			Interval:                int64(started.Interval),
		},
	})
}

func (h *Handler) get(ctx context.Context, s *zen.Session) error {
	if _, err := s.GetPrincipal(); err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.V2CliGetDeviceLoginRequestBody](s)
	if err != nil {
		return err
	}
	view, err := h.Devices.Get(ctx, req.UserCode)
	if err != nil {
		return err
	}
	var deviceName *string
	if view.DeviceName != "" {
		deviceName = &view.DeviceName
	}
	return s.JSON(http.StatusOK, openapi.V2CliGetDeviceLoginResponseBody{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2CliGetDeviceLoginResponseData{
			UserCode:              view.UserCode,
			DeviceName:            deviceName,
			Status:                view.Status,
			ExpiresAt:             view.ExpiresAt,
			WorkosVerificationUri: view.WorkOSVerificationURI,
		},
	})
}

func (h *Handler) approve(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.V2CliApproveDeviceLoginRequestBody](s)
	if err != nil {
		return err
	}
	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	if err := h.Devices.Approve(ctx, clidevice.ApproveRequest{
		Principal:   p,
		UserCode:    req.UserCode,
		Name:        name,
		Permissions: req.Permissions,
	}); err != nil {
		return err
	}
	return s.JSON(http.StatusOK, openapi.V2CliApproveDeviceLoginResponseBody{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

func (h *Handler) deny(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.V2CliDenyDeviceLoginRequestBody](s)
	if err != nil {
		return err
	}
	if err := h.Devices.Deny(ctx, p, req.UserCode); err != nil {
		return err
	}
	return s.JSON(http.StatusOK, openapi.V2CliDenyDeviceLoginResponseBody{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}

func (h *Handler) poll(ctx context.Context, s *zen.Session) error {
	req, err := zen.BindBody[openapi.V2CliPollDeviceLoginRequestBody](s)
	if err != nil {
		return err
	}
	result, err := h.Devices.Poll(ctx, req.LoginId, s.Location(), s.UserAgent())
	if err != nil {
		return err
	}
	data := openapi.V2CliPollDeviceLoginResponseData{
		Status:   string(result.Status),
		Interval: int64(result.Interval),
		Detail:   nil,
		Key:      nil,
		KeyId:    nil,
	}
	if result.Detail != "" {
		data.Detail = &result.Detail
	}
	if result.Key != nil {
		data.Key = result.Key
		data.KeyId = &result.KeyID
	}
	return s.JSON(http.StatusOK, openapi.V2CliPollDeviceLoginResponseBody{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}

type StartHandler struct{ Handler }

func (h *StartHandler) Method() string { return http.MethodPost }
func (h *StartHandler) Path() string   { return "/v2/cli.startDeviceLogin" }
func (h *StartHandler) Handle(ctx context.Context, s *zen.Session) error {
	return h.start(ctx, s)
}

type GetHandler struct{ Handler }

func (h *GetHandler) Method() string { return http.MethodPost }
func (h *GetHandler) Path() string   { return "/v2/cli.getDeviceLogin" }
func (h *GetHandler) Handle(ctx context.Context, s *zen.Session) error {
	return h.get(ctx, s)
}

type ApproveHandler struct{ Handler }

func (h *ApproveHandler) Method() string { return http.MethodPost }
func (h *ApproveHandler) Path() string   { return "/v2/cli.approveDeviceLogin" }
func (h *ApproveHandler) Handle(ctx context.Context, s *zen.Session) error {
	return h.approve(ctx, s)
}

type DenyHandler struct{ Handler }

func (h *DenyHandler) Method() string { return http.MethodPost }
func (h *DenyHandler) Path() string   { return "/v2/cli.denyDeviceLogin" }
func (h *DenyHandler) Handle(ctx context.Context, s *zen.Session) error {
	return h.deny(ctx, s)
}

type PollHandler struct{ Handler }

func (h *PollHandler) Method() string { return http.MethodPost }
func (h *PollHandler) Path() string   { return "/v2/cli.pollDeviceLogin" }
func (h *PollHandler) Handle(ctx context.Context, s *zen.Session) error {
	return h.poll(ctx, s)
}
