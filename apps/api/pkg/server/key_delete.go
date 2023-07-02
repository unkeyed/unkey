package server

import (
	"errors"
	"fmt"
	"github.com/chronark/unkey/apps/api/pkg/database"
	"net/http"

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
		return c.Status(400).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: err.Error(),
		})
	}

	err = s.validator.Struct(req)
	if err != nil {
		return c.Status(400).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: err.Error(),
		})
	}

	authHash, err := getKeyHash(c.Get("Authorization"))
	if err != nil {
		return err
	}

	authKey, err := s.db.GetKeyByHash(ctx, authHash)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
				Code:  UNAUTHORIZED,
				Error: "unauthorized",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: err.Error(),
		})
	}

	if authKey.ForWorkspaceId == "" {
		return c.Status(400).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: "wrong key type",
		})
	}

	key, err := s.db.GetKeyById(ctx, req.KeyId)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return c.Status(http.StatusNotFound).JSON(ErrorResponse{
				Code:  NOT_FOUND,
				Error: fmt.Sprintf("key %s does not exist", req.KeyId),
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: err.Error(),
		})
	}

	if key.WorkspaceId != authKey.ForWorkspaceId {
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:  UNAUTHORIZED,
			Error: "access to workspace denied",
		})
	}

	err = s.db.DeleteKey(ctx, key.Id)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to delete key %s", err.Error()),
		})
	}
	if s.kafka != nil {

		err := s.kafka.ProduceKeyDeletedEvent(ctx, key.Id, key.Hash)
		if err != nil {
			s.logger.Error("unable to emit keyDeletedEvent", zap.Error(err))
		}
	}
	return c.JSON(DeleteKeyResponse{})
}
