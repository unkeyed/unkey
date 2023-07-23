package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/kafka"
	"go.uber.org/zap"
)

// nullish is a wrapper to allow a value to be optional or null
// It's optional if `Defined` is false. In that case you should not check `Value`
// The idea is to represent the following typescript type `T | undefined | null`
type nullish[T any] struct {
	Defined bool
	Value   *T
}

func (m *nullish[T]) UnmarshalJSON(data []byte) error {
	m.Defined = true
	return json.Unmarshal(data, &m.Value)
}

type UpdateKeyRequest struct {
	KeyId     string                  `json:"keyId" validate:"required"`
	Name      nullish[string]         `json:"name"`
	OwnerId   nullish[string]         `json:"ownerId"`
	Meta      nullish[map[string]any] `json:"meta"`
	Expires   nullish[int64]          `json:"expires"`
	Ratelimit nullish[struct {
		Type           string `json:"type" validate:"required"`
		Limit          int64  `json:"limit" validate:"required"`
		RefillRate     int64  `json:"refillRate" validate:"required"`
		RefillInterval int64  `json:"refillInterval" validate:"required"`
	}] `json:"ratelimit"`
	Remaining nullish[int64] `json:"remaining"`
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

	s.logger.Info("req", zap.Any("req", req))

	err = c.BodyParser(&req)
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
	if req.Expires.Defined && req.Expires.Value != nil && *req.Expires.Value > 0 && *req.Expires.Value < time.Now().UnixMilli() {
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

	if req.Name.Defined {
		if req.Name.Value != nil {
			key.Name = *req.Name.Value
		} else {
			key.Name = ""
		}
	}

	if req.OwnerId.Defined {
		if req.OwnerId.Value != nil {
			key.OwnerId = *req.OwnerId.Value
		} else {
			key.OwnerId = ""
		}
	}
	if req.Meta.Defined {
		if req.Meta.Value != nil {
			key.Meta = *req.Meta.Value
		} else {
			key.Meta = nil
		}
	}
	if req.Expires.Defined {
		if req.Expires.Value != nil {
			key.Expires = time.UnixMilli(*req.Expires.Value)
		} else {
			key.Expires = time.Time{}
		}
	}
	if req.Ratelimit.Defined {
		if req.Ratelimit.Value != nil {
			key.Ratelimit = &entities.Ratelimit{
				Type:           req.Ratelimit.Value.Type,
				Limit:          req.Ratelimit.Value.Limit,
				RefillRate:     req.Ratelimit.Value.RefillRate,
				RefillInterval: req.Ratelimit.Value.RefillInterval,
			}
		} else {
			key.Ratelimit = nil
		}
	}
	if req.Remaining.Defined {
		if req.Remaining.Value != nil {
			key.Remaining.Enabled = true
			key.Remaining.Remaining = *req.Remaining.Value
		} else {
			key.Remaining.Enabled = false
			key.Remaining.Remaining = 0
		}
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
