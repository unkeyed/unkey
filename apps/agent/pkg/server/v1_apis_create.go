package server

import (
	"github.com/gofiber/fiber/v2"
	httpErrors "github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/services/apis"
)

type CreateApiRequest struct {
	Name string `json:"name" validate:"required"`
}

type CreateApiResponse struct {
	ApiId string `json:"apiId"`
}

func (s *Server) v1CreateApi(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.v1.apis.createApi")
	defer span.End()
	req := CreateApiRequest{}

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

	created, err := s.apiService.CreateApi(ctx, apis.CreateApiRequest{
		Name:        req.Name,
		WorkspaceId: authorizedWorkspaceId,
	})
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.INTERNAL_SERVER_ERROR, err.Error())
	}

	return c.JSON(CreateApiResponse{
		ApiId: created.ApiId,
	})
}
