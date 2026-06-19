package handler

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ProjectsCreateProjectRequestBody
	Response = openapi.V2ProjectsCreateProjectResponseBody
)

type Handler struct {
	CtrlClient ctrl.ProjectServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/projects.createProject"
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

	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Project,
		ResourceID:   "*",
		Action:       rbac.CreateProject,
	}))
	if err != nil {
		return err
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	ctrlResp, err := h.CtrlClient.CreateProject(ctx, &ctrlv1.CreateProjectRequest{
		WorkspaceId: principal.WorkspaceID,
		Name:        req.Name,
		Slug:        req.Slug,
		Actor:       actor,
	})
	if err != nil {
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Project.Duplicate.URN()),
				fault.Internal("project slug already exists"),
				fault.Public(fmt.Sprintf("A project with slug '%s' already exists in this workspace.", req.Slug)),
			)
		}
		return ctrlclient.HandleError(err, "create project")
	}

	projectID := ctrlResp.GetId()

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2ProjectsCreateProjectResponseData{
			Id: projectID,
		},
	})
}
