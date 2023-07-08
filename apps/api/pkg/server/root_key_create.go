package server

import (
	"crypto/subtle"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/hash"
	"github.com/unkeyed/unkey/apps/api/pkg/kafka"
	"github.com/unkeyed/unkey/apps/api/pkg/keys"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
)

type CreateRootKeyRequest struct {
	Name    string `json:"name"`
	Expires int64  `json:"expires"`

	// ForWorkspaceId is used internally when the frontend wants to create a new root key.
	// Therefore we might not want to add this field to our docs.
	ForWorkspaceId string `json:"forWorkspaceId" validate:"required"`
}

type CreateRootKeyResponse struct {
	Key   string `json:"key"`
	KeyId string `json:"keyId"`
}

func (s *Server) createRootKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.createRootKey")
	defer span.End()

	appToken := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	if subtle.ConstantTimeCompare([]byte(s.unkeyAppAuthToken), []byte(appToken)) == 0 {
		return c.Status(http.StatusUnauthorized).JSON(
			ErrorResponse{
				Code:  UNAUTHORIZED,
				Error: "unauthorized",
			})
	}

	req := CreateRootKeyRequest{}
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

	if req.Expires > 0 && req.Expires < time.Now().UnixMilli() {
		return c.Status(http.StatusBadRequest).JSON(
			ErrorResponse{
				Code:  BAD_REQUEST,
				Error: "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?",
			})
	}

	keyValue, err := keys.NewV1Key("unkey", 16)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: err.Error(),
		})
	}
	separatorIndex := strings.Index(keyValue, "_")
	keyHash := hash.Sha256(keyValue)

	newKey := entities.Key{
		Id:          uid.Key(),
		ApiId:       s.unkeyApiId,
		WorkspaceId: s.unkeyWorkspaceId,
		Name:        req.Name,
		Hash:        keyHash,
		Start:       keyValue[0 : separatorIndex+4],
		CreatedAt:   time.Now(),
		Ratelimit: &entities.Ratelimit{
			Type:           "fast",
			Limit:          100,
			RefillRate:     10,
			RefillInterval: 1000,
		},
		ForWorkspaceId: req.ForWorkspaceId,
	}
	if req.Expires > 0 {
		newKey.Expires = time.UnixMilli(req.Expires)
	}

	err = s.db.CreateKey(ctx, newKey)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to store key: %s", err.Error()),
		})
	}
	if s.kafka != nil {

		go func() {
			err := s.kafka.ProduceKeyEvent(ctx, kafka.KeyCreated, newKey.Id, newKey.Hash)
			if err != nil {
				s.logger.Error("unable to emit new key event to kafka", zap.Error(err))
			}
		}()
	}
	return c.JSON(CreateRootKeyResponse{
		Key:   keyValue,
		KeyId: newKey.Id,
	})
}
