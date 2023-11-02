package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	apisv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/apis/v1"
)

type FindApiRequestV1 struct {
	ApiId string `json:"apiId" validate:"required"`
}

type GetApiResponseV1 struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	WorkspaceId string   `json:"workspaceId"`
	IpWhitelist []string `json:"ipWhitelist,omitempty"`
}

func (s *Server) v1FindApi(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.getApi")
	defer span.End()

	req := FindApiRequestV1{
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
		return fromServiceError(c, err)
	}
	if !auth.IsRootKey {
		return newHttpError(c, UNAUTHORIZED, "root key required")
	}
	api, err := s.apiService.FindApi(ctx, &apisv1.FindApiRequest{
		ApiId: req.ApiId,
	})
	if err != nil {
		return fromServiceError(c, err)
	}

	return c.JSON(GetApiResponseV1{
		Id:          api.Api.ApiId,
		Name:        api.Api.Name,
		WorkspaceId: api.Api.WorkspaceId,
		IpWhitelist: api.Api.IpWhitelist,
	})
}
