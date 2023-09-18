package server

import (
	"github.com/gofiber/fiber/v2"
	httpErrors "github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
)

type RemoveApiRequest struct {
	ApiId string `json:"apiId" validate:"required"`
}

type RemoveApiResponse struct {
}

func (s *Server) v1RemoveApi(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.v1.apis.createApi")
	defer span.End()
	req := RemoveApiRequest{}

	err := c.BodyParser(&req)
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.BAD_REQUEST, err.Error())
	}

	err = s.validator.Struct(req)
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.BAD_REQUEST, err.Error())
	}

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c.Get("Authorization"))
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.UNAUTHORIZED, err.Error())
	}

	api, found, err := s.db.FindApi(ctx, req.ApiId)
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.INTERNAL_SERVER_ERROR, err.Error())
	}
	if !found {
		return httpErrors.NewHttpError(c, httpErrors.NOT_FOUND, "api not found")
	}
	if api.WorkspaceId != authorizedWorkspaceId {
		return httpErrors.NewHttpError(c, httpErrors.UNAUTHORIZED, "access to workspace denied")
	}
	_, err = s.apiService.RemoveApi(ctx, apis.RemoveApiRequest{ApiId: req.ApiId})
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.INTERNAL_SERVER_ERROR, err.Error())
	}

	return c.JSON(RemoveApiResponse{})
}
