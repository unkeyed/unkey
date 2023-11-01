package server

import (
	"fmt"

	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"

	"github.com/gofiber/fiber/v2"
)

type RemoveKeyRequestV1 struct {
	KeyId string `json:"keyId" validate:"required"`
}

type RemoveKeyResponseV1 struct {
}

func (s *Server) v1RemoveKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.v1RemoveKey")
	defer span.End()
	req := RemoveKeyRequestV1{}

	err := c.BodyParser(&req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}
	err = s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}

	auth, err := s.authorizeKey(ctx, c)
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}
	if !auth.IsRootKey {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "root key required")
	}
	key, found, err := cache.WithCache(s.keyCache, s.db.FindKeyById)(ctx, req.KeyId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find key: %s", err.Error()))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find key: %s", req.KeyId))
	}
	if key.WorkspaceId != auth.AuthorizedWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "access to workspace denied")
	}

	_, err = s.keyService.SoftDeleteKey(ctx, &authenticationv1.SoftDeleteKeyRequest{
		KeyId: key.KeyId,
	})
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to delete key: %s", err.Error()))
	}

	return c.JSON(RemoveKeyResponseV1{})
}
