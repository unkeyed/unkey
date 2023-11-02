package server

import (
	"fmt"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"

	"github.com/gofiber/fiber/v2"
)

type DeleteKeyRequestV1 struct {
	KeyId string `json:"keyId" validate:"required"`
}

type DeleteKeyResponseV1 struct {
}

func (s *Server) v1DeleteKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.v1DeleteKey")
	defer span.End()

	req := DeleteKeyRequestV1{}
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

	_, err = s.keyService.SoftDeleteKey(ctx, &authenticationv1.SoftDeleteKeyRequest{
		KeyId:                 req.KeyId,
		AuthorizedWorkspaceId: auth.AuthorizedWorkspaceId,
	})
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to delete key: %s", err.Error()))
	}

	return c.JSON(DeleteKeyResponseV1{})
}
