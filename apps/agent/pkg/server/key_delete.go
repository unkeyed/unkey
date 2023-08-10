package server

import (
	"fmt"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/kafka"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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

	authHash, err := getKeyHash(c.Get("Authorization"))
	if err != nil {
		return err
	}

	authKey, found, err := s.db.FindKeyByHash(ctx, authHash)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find key: %s", err.Error()))

	}
	if !found {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "")
	}

	if authKey.ForWorkspaceId == "" {
		return errors.NewHttpError(c, errors.INVALID_KEY_TYPE, "you need to use a root key")

	}

	key, found, err := s.db.FindKeyById(ctx, req.KeyId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find key: %s", err.Error()))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find key: %s", req.KeyId))
	}
	if key.WorkspaceId != authKey.ForWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "access to workspace denied")
	}

	err = s.db.DeleteKey(ctx, key.Id)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to delete key: %s", err.Error()))
	}
	if s.kafka != nil {
		err := s.kafka.ProduceKeyEvent(ctx, kafka.KeyDeleted, key.Id, key.Hash)
		if err != nil {
			s.logger.Error("unable to emit keyDeletedEvent", zap.Error(err))
		}
	}
	return c.JSON(DeleteKeyResponse{})
}
