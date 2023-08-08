package server

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/entities"
	"github.com/unkeyed/unkey/apps/api/pkg/errors"
	"github.com/unkeyed/unkey/apps/api/pkg/hash"
	"github.com/unkeyed/unkey/apps/api/pkg/kafka"
	"github.com/unkeyed/unkey/apps/api/pkg/keys"
	"github.com/unkeyed/unkey/apps/api/pkg/uid"
	"go.uber.org/zap"
	"time"
)

type CreateKeyRequest struct {
	ApiId      string         `json:"apiId" validate:"required"`
	Prefix     string         `json:"prefix"`
	Name       string         `json:"name"`
	ByteLength int            `json:"byteLength"`
	OwnerId    string         `json:"ownerId"`
	Meta       map[string]any `json:"meta"`
	Expires    int64          `json:"expires"`
	Ratelimit  *struct {
		Type           string `json:"type"`
		Limit          int32  `json:"limit"`
		RefillRate     int32  `json:"refillRate"`
		RefillInterval int32  `json:"refillInterval"`
	} `json:"ratelimit"`
	// ForWorkspaceId is used internally when the frontend wants to create a new root key.
	// Therefore we might not want to add this field to our docs.
	ForWorkspaceId string `json:"forWorkspaceId"`

	// How often this key may be used
	// `undefined`, `0` or negative to disable
	Remaining int32 `json:"remaining,omitempty"`
}

type CreateKeyResponse struct {
	Key   string `json:"key"`
	KeyId string `json:"keyId"`
}

func (s *Server) createKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.createKey")
	defer span.End()

	req := CreateKeyRequest{
		// These act as default
		ByteLength: 16,
	}
	err := c.BodyParser(&req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, fmt.Sprintf("unable to parse body: %s", err.Error()))
	}

	err = s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, fmt.Sprintf("unable to validate body: %s", err.Error()))
	}

	if req.Expires > 0 && req.Expires < time.Now().UnixMilli() {
		return errors.NewHttpError(c, errors.BAD_REQUEST, "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?")
	}

	authHash, err := getKeyHash(c.Get("Authorization"))
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}

	authKey, found, err := s.db.FindKeyByHash(ctx, authHash)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find key: %w", err))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, "unable to find key")

	}

	if authKey.ForWorkspaceId == "" {
		return errors.NewHttpError(c, errors.INVALID_KEY_TYPE, "a root key is required")
	}

	api, found, err := s.db.FindApi(ctx, req.ApiId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find api: %w", err.Error()))

	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find api: %s", req.ApiId))
	}
	if api.WorkspaceId != authKey.ForWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "access to workspace denied")

	}

	if api.AuthType != entities.AuthTypeKey || api.KeyAuthId == "" {
		return errors.NewHttpError(c, errors.BAD_REQUEST, "api is not setup to handle api keys")

	}

	keyValue, err := keys.NewV1Key(req.Prefix, req.ByteLength)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to create new key: %s", err.Error()))

	}
	// how many chars to store, this includes the prefix, delimiter and the first 4 characters of the key
	startLength := len(req.Prefix) + 5
	keyHash := hash.Sha256(keyValue)

	newKey := entities.Key{
		Id:          uid.Key(),
		KeyAuthId:   api.KeyAuthId,
		WorkspaceId: authKey.ForWorkspaceId,
		Name:        req.Name,
		Hash:        keyHash,
		Start:       keyValue[:startLength],
		OwnerId:     req.OwnerId,
		Meta:        req.Meta,
		CreatedAt:   time.Now(),
	}
	if req.Expires > 0 {
		newKey.Expires = time.UnixMilli(req.Expires)
	}
	if req.Remaining > 0 {
		remaining := req.Remaining
		newKey.Remaining = &remaining
	}
	if req.Ratelimit != nil {
		newKey.Ratelimit = &entities.Ratelimit{
			Type:           req.Ratelimit.Type,
			Limit:          req.Ratelimit.Limit,
			RefillRate:     req.Ratelimit.RefillRate,
			RefillInterval: req.Ratelimit.RefillInterval,
		}
	}

	err = s.db.CreateKey(ctx, newKey)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to store key: %s", err.Error()))
	}
	if s.kafka != nil {

		go func() {
			err := s.kafka.ProduceKeyEvent(ctx, kafka.KeyCreated, newKey.Id, newKey.Hash)
			if err != nil {
				s.logger.Error("unable to emit new key event to kafka", zap.Error(err))
			}
		}()
	}

	return c.JSON(CreateKeyResponse{
		Key:   keyValue,
		KeyId: newKey.Id,
	})
}
