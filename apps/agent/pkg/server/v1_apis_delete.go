package server

import (
	"github.com/gofiber/fiber/v2"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
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

	err := s.parseAndValidate(c, &req)
	if err != nil {
		return newHttpError(c, BAD_REQUEST, err.Error())
	}

	auth, err := s.authorizeKey(ctx, c)
	if err != nil {
		return newHttpError(c, UNAUTHORIZED, err.Error())
	}
	if !auth.IsRootKey {
		return newHttpError(c, UNAUTHORIZED, "root key required")
	}

	_, err = s.apiService.DeleteApi(ctx, &apisv1.DeleteApiRequest{ApiId: req.ApiId, AuthorizedWorkspaceId: auth.AuthorizedWorkspaceId})
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, err.Error())
	}

	return c.JSON(DeleteApiResponse{})
}
