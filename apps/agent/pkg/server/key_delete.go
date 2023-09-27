package server

import (
	"fmt"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"

	"github.com/gofiber/fiber/v2"
)

type DeleteKeyRequest struct {
	KeyId string `json:"keyId" validate:"required"`
}

type DeleteKeyResponse struct {
}

func (s *Server) deleteKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.deleteKey")
	defer span.End()
	req := DeleteKeyRequest{}
	err := c.ParamsParser(&req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}

	err = s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c)
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}
	key, found, err := cache.WithCache(s.keyCache, s.db.FindKeyById)(ctx, req.KeyId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find key: %s", err.Error()))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find key: %s", req.KeyId))
	}
	if key.WorkspaceId != authorizedWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "access to workspace denied")
	}

	err = s.db.DeleteKey(ctx, key.Id)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to delete key: %s", err.Error()))
	}

	s.events.EmitKeyEvent(ctx, events.KeyEvent{
		Type: events.KeyDeleted,
		Key: events.Key{
			Id:   key.Id,
			Hash: key.Hash,
		},
	})

	return c.JSON(DeleteKeyResponse{})
}
