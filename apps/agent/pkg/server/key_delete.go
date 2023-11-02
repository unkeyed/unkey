package server

import (
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"

	"github.com/gofiber/fiber/v2"
)

func (s *Server) deleteKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.deleteKey")
	defer span.End()
	req := DeleteKeyRequestV1{}
	err := c.ParamsParser(&req)
	if err != nil {
		return newHttpError(c, BAD_REQUEST, err.Error())
	}

	err = s.validator.Struct(req)
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

	_, err = s.keyService.SoftDeleteKey(ctx, &authenticationv1.SoftDeleteKeyRequest{KeyId: req.KeyId, AuthorizedWorkspaceId: auth.AuthorizedWorkspaceId})
	if err != nil {
		return fromServiceError(c, err)
	}

	return c.JSON(DeleteKeyResponseV1{})
}
