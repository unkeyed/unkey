package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

type GetApiRequest struct {
	ApiId string `json:"apiId" validate:"required"`
}

type GetApiResponse struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	WorkspaceId string   `json:"workspaceId"`
	IpWhitelist []string `json:"ipWhitelist,omitempty"`
}

func (s *Server) getApi(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.getApi")
	defer span.End()

	req := GetApiRequest{
		ApiId: c.Params("apiId"),
	}

	err := s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, fmt.Sprintf("unable to validate request: %s", err.Error()))
	}

	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}

	auth, err := s.authorizeKey(ctx, c)
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}
	if !auth.IsRootKey {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "root key required")
	}

	api, found, err := s.db.FindApi(ctx, req.ApiId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find api: %s", err.Error()))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find api: %s", req.ApiId))
	}

	if api.WorkspaceId != auth.AuthorizedWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "access to workspace denied")
	}

	return c.JSON(GetApiResponse{
		Id:          api.ApiId,
		Name:        api.Name,
		WorkspaceId: api.WorkspaceId,
		IpWhitelist: api.IpWhitelist,
	})
}
