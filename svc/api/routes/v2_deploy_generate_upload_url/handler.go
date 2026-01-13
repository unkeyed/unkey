package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeployGenerateUploadUrlRequestBody
	Response = openapi.V2DeployGenerateUploadUrlResponseBody
)

type Handler struct {
	Logger     logging.Logger
	DB         db.Database
	Keys       keys.KeyService
	CtrlClient ctrlv1connect.BuildServiceClient
}

func (h *Handler) Path() string {
	return "/v2/deploy.generateUploadUrl"
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	_, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	if req.ProjectId == "" {
		return fault.New("projectId is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("projectId is required."),
		)
	}

	// nolint: exhaustruct // optional proto fields, only setting whats provided
	ctrlReq := &ctrlv1.GenerateUploadURLRequest{
		UnkeyProjectId: req.ProjectId,
	}

	connectReq := connect.NewRequest(ctrlReq)

	ctrlResp, err := h.CtrlClient.GenerateUploadURL(ctx, connectReq)
	if err != nil {
		return ctrlclient.HandleError(err, "generate upload URL")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2DeployGenerateUploadUrlResponseData{
			UploadUrl:        ctrlResp.Msg.GetUploadUrl(),
			BuildContextPath: ctrlResp.Msg.GetBuildContextPath(),
			ExpiresIn:        ctrlResp.Msg.GetExpiresIn(),
		},
	})
}
