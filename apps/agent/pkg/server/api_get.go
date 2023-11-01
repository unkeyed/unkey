package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
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
		return newHttpError(c, BAD_REQUEST, fmt.Sprintf("unable to validate request: %s", err.Error()))
	}

	if err != nil {
		return newHttpError(c, UNAUTHORIZED, err.Error())
	}

	auth, err := s.authorizeKey(ctx, c)
	if err != nil {
		return newHttpError(c, UNAUTHORIZED, err.Error())
	}
	if !auth.IsRootKey {
		return newHttpError(c, UNAUTHORIZED, "root key required")
	}

	api, found, err := s.db.FindApi(ctx, req.ApiId)
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find api: %s", err.Error()))
	}
	if !found {
		return newHttpError(c, NOT_FOUND, fmt.Sprintf("unable to find api: %s", req.ApiId))
	}

	if api.WorkspaceId != auth.AuthorizedWorkspaceId {
		return newHttpError(c, UNAUTHORIZED, "access to workspace denied")
	}

	return c.JSON(GetApiResponse{
		Id:          api.ApiId,
		Name:        api.Name,
		WorkspaceId: api.WorkspaceId,
		IpWhitelist: api.IpWhitelist,
	})
}
