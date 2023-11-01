package server

import (
	"github.com/gofiber/fiber/v2"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	httpErrors "github.com/unkeyed/unkey/apps/agent/pkg/errors"
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

	auth, err := s.authorizeKey(ctx, c)
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.UNAUTHORIZED, err.Error())
	}
	if !auth.IsRootKey {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "root key required")
	}

	created, err := s.apiService.CreateApi(ctx, &apisv1.CreateApiRequest{
		Name:        req.Name,
		WorkspaceId: auth.AuthorizedWorkspaceId,
	})
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.INTERNAL_SERVER_ERROR, err.Error())
	}

	return c.JSON(CreateApiResponse{
		ApiId: created.ApiId,
	})
}
