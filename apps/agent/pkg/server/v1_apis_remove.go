package server

import (
	"github.com/gofiber/fiber/v2"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	httpErrors "github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

type DeleteApiRequest struct {
	ApiId string `json:"apiId" validate:"required"`
}

type DeleteApiResponse struct {
}

func (s *Server) v1DeleteApi(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.v1.apis.createApi")
	defer span.End()
	req := DeleteApiRequest{}

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

	api, found, err := s.db.FindApi(ctx, req.ApiId)
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.INTERNAL_SERVER_ERROR, err.Error())
	}
	if !found {
		return httpErrors.NewHttpError(c, httpErrors.NOT_FOUND, "api not found")
	}
	if api.WorkspaceId != auth.AuthorizedWorkspaceId {
		return httpErrors.NewHttpError(c, httpErrors.UNAUTHORIZED, "access to workspace denied")
	}
	_, err = s.apiService.DeleteApi(ctx, &apisv1.DeleteApiRequest{ApiId: req.ApiId})
	if err != nil {
		return httpErrors.NewHttpError(c, httpErrors.INTERNAL_SERVER_ERROR, err.Error())
	}

	return c.JSON(DeleteApiResponse{})
}
