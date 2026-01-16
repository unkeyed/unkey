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
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeployGetDeploymentRequestBody
	Response = openapi.V2DeployGetDeploymentResponseBody
)

type Handler struct {
	Logger     logging.Logger
	DB         db.Database
	Keys       keys.KeyService
	CtrlClient ctrlv1connect.DeploymentServiceClient
}

func (h *Handler) Path() string {
	return "/v2/deploy.getDeployment"
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.ReadDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   req.ProjectId,
			Action:       rbac.ReadDeployment,
		}),
	)))
	if err != nil {
		return err
	}

	// Verify project belongs to the authenticated workspace
	project, err := db.Query.FindProjectById(ctx, h.DB.RO(), req.ProjectId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("project not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project not found"),
				fault.Public("The requested project does not exist or has been deleted."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to find project"))
	}
	if project.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"),
			fault.Public("The requested project does not exist or has been deleted."),
		)
	}

	ctrlReq := &ctrlv1.GetDeploymentRequest{
		DeploymentId: req.DeploymentId,
	}
	connectReq := connect.NewRequest(ctrlReq)

	ctrlResp, err := h.CtrlClient.GetDeployment(ctx, connectReq)
	if err != nil {
		return ctrlclient.HandleError(err, "get deployment")
	}

	deployment := ctrlResp.Msg.GetDeployment()

	// Transform status enum to string
	statusStr := deploymentStatusToString(deployment.GetStatus())

	// Transform steps
	var steps *[]openapi.V2DeployDeploymentStep
	if deployment.GetSteps() != nil {
		stepsSlice := make([]openapi.V2DeployDeploymentStep, len(deployment.GetSteps()))
		for i, protoStep := range deployment.GetSteps() {
			step := openapi.V2DeployDeploymentStep{
				ErrorMessage: nil,
				CreatedAt:    nil,
				Message:      nil,
				Status:       nil,
			}

			if protoStep.GetStatus() != "" {
				status := protoStep.GetStatus()
				step.Status = &status
			}
			if protoStep.GetMessage() != "" {
				message := protoStep.GetMessage()
				step.Message = &message
			}
			if protoStep.GetErrorMessage() != "" {
				errMessage := protoStep.GetErrorMessage()
				step.ErrorMessage = &errMessage
			}
			if protoStep.GetCreatedAt() != 0 {
				createdAt := protoStep.GetCreatedAt()
				step.CreatedAt = &createdAt
			}
			stepsSlice[i] = step
		}
		steps = &stepsSlice
	}

	responseData := openapi.V2DeployGetDeploymentResponseData{
		Id:           deployment.GetId(),
		Status:       openapi.V2DeployGetDeploymentResponseDataStatus(statusStr),
		Steps:        nil,
		ErrorMessage: nil,
		Hostnames:    nil,
	}

	if deployment.GetErrorMessage() != "" {
		errorMessage := deployment.GetErrorMessage()
		responseData.ErrorMessage = &errorMessage
	}

	if len(deployment.GetHostnames()) > 0 {
		hostnames := deployment.GetHostnames()
		responseData.Hostnames = &hostnames
	}

	if steps != nil {
		responseData.Steps = steps
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}

func deploymentStatusToString(status ctrlv1.DeploymentStatus) string {
	switch status {
	case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED:
		return "UNSPECIFIED"
	case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING:
		return "PENDING"
	case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_BUILDING:
		return "BUILDING"
	case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_DEPLOYING:
		return "DEPLOYING"
	case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_NETWORK:
		return "NETWORK"
	case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY:
		return "READY"
	case ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED:
		return "FAILED"
	default:
		return "UNSPECIFIED"
	}
}
