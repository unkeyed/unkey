package server

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/kafka"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type UpdateKeyRequest struct {
	KeyId     string          `json:"keyId" validate:"required"`
	Name      *string         `json:"name,omitempty"`
	OwnerId   *string         `json:"ownerId,omitempty"`
	Meta      *map[string]any `json:"meta,omitempty"`
	Expires   *int64          `json:"expires,omitempty"`
	Ratelimit *struct {
		Type           string `json:"type"`
		Limit          int64  `json:"limit"`
		RefillRate     int64  `json:"refillRate"`
		RefillInterval int64  `json:"refillInterval"`
	} `json:"ratelimit,omitempty"`

	// How often this key may be used
	// `undefined`, `0` or negative to disable
	Remaining *int64 `json:"remaining,omitempty"`
}

type UpdateKeyResponse struct{}

func (s *Server) updateKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.updateKey")
	defer span.End()

	req := UpdateKeyRequest{
		KeyId: c.Params("keyId"),
	}
	err := c.BodyParser(&req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to parse body: %s", err.Error()),
		})
	}

	err = s.validator.Struct(req)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to validate body: %s", err.Error()),
		})
	}

	s.logger.Info("updating key", zap.Any("req", req))
	if req.Expires != nil && *req.Expires > 0 && *req.Expires < time.Now().UnixMilli() {
		return c.Status(http.StatusBadRequest).JSON(
			ErrorResponse{
				Code:  BAD_REQUEST,
				Error: "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?",
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
			Error: fmt.Sprintf("unable to find key: %s", err.Error()),
		})
	}

	if authKey.ForWorkspaceId == "" {
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: "wrong key type",
		})
	}

	key, err := s.db.GetKeyById(ctx, req.KeyId)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
				Code:  BAD_REQUEST,
				Error: "wrong keyId",
			})
		}
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to find key: %s", err.Error()),
		})
	}
	if key.WorkspaceId != authKey.ForWorkspaceId {
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:  UNAUTHORIZED,
			Error: "access to workspace denied",
		})
	}

	s.logger.Info("found key", zap.Any("key", key))

	if req.Name != nil {
		key.Name = *req.Name
	}

	if req.OwnerId != nil {
		key.OwnerId = *req.OwnerId
	}
	if req.Meta != nil {
		key.Meta = *req.Meta
	}
	if req.Expires != nil {
		key.Expires = time.UnixMilli(*req.Expires)
	}
	if req.Ratelimit != nil {
		key.Ratelimit = &entities.Ratelimit{
			Type:           req.Ratelimit.Type,
			Limit:          req.Ratelimit.Limit,
			RefillRate:     req.Ratelimit.RefillRate,
			RefillInterval: req.Ratelimit.RefillInterval,
		}
	}
	if req.Remaining != nil {
		key.Remaining.Enabled = true
		key.Remaining.Remaining = *req.Remaining
	}

	err = s.db.UpdateKey(ctx, key)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to write key: %s", err.Error()),
		})
	}
	if s.kafka != nil {

		go func() {
			err := s.kafka.ProduceKeyEvent(ctx, kafka.KeyUpdated, key.Id, key.Hash)
			if err != nil {
				s.logger.Error("unable to emit key event to kafka", zap.Error(err))
			}
		}()
	}

	return c.JSON(UpdateKeyResponse{})
}
