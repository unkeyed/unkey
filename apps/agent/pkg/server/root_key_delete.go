package server

import (
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"

	"github.com/gofiber/fiber/v2"
)

type DeleteRootKeyRequest struct {
	KeyId string `json:"keyId" validate:"required"`
}

type DeleteRootKeyResponse struct {
}

func (s *Server) deleteRootKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.deleteKey")
	defer span.End()
	req := DeleteRootKeyRequest{}
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
	if key.ForWorkspaceId == nil || key.GetForWorkspaceId() != auth.AuthorizedWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "access to workspace denied")
	}

	err = s.db.SoftDeleteKey(ctx, key.KeyId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to delete key: %s", err.Error()))
	}

	s.events.EmitKeyEvent(ctx, events.KeyEvent{
		Type: events.KeyDeleted,
		Key: events.Key{
			Id:   key.KeyId,
			Hash: key.Hash,
		},
	})

	return c.JSON(DeleteRootKeyResponse{})
}
