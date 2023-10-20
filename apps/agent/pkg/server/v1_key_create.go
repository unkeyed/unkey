package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	keysv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/keys/v1"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

type CreateKeyRequest struct {
	ApiId      string         `json:"apiId" validate:"required"`
	Prefix     string         `json:"prefix"`
	Name       string         `json:"name"`
	ByteLength int32          `json:"byteLength"`
	OwnerId    string         `json:"ownerId"`
	Meta       map[string]any `json:"meta"`
	Expires    int64          `json:"expires"`
	Ratelimit  *struct {
		Type           string `json:"type" validate:"required,oneof=fast consistent"`
		Limit          int32  `json:"limit" validate:"required"`
		RefillRate     int32  `json:"refillRate" validate:"required"`
		RefillInterval int32  `json:"refillInterval"  validate:"required"`
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

func (s *Server) v1CreateKey(c *fiber.Ctx) error {
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

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c)
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}

	api, found, err := cache.WithCache(s.apiCache, s.db.FindApi)(ctx, req.ApiId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find api: %s", err.Error()))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find api: %s", req.ApiId))
	}
	if api.WorkspaceId != authorizedWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "access to workspace denied")

	}

	if api.AuthType != entities.AuthTypeKey || api.KeyAuthId == "" {
		return errors.NewHttpError(c, errors.BAD_REQUEST, "api is not setup to handle api keys")

	}

	createKeyReq := &keysv1.CreateKeyRequest{
		WorkspaceId: api.WorkspaceId,
		KeyAuthId:   api.KeyAuthId,
	}
	if req.Expires > 0 {
		createKeyReq.Expires = &req.Expires
	}
	if req.ByteLength > 0 {
		createKeyReq.ByteLength = util.Pointer(req.ByteLength)
	}
	if req.Name != "" {
		createKeyReq.Name = util.Pointer(req.Name)
	}
	if req.OwnerId != "" {
		createKeyReq.OwnerId = util.Pointer(req.OwnerId)
	}
	if req.Prefix != "" {
		createKeyReq.Prefix = util.Pointer(req.Prefix)
	}
	if req.Meta != nil {
		b, err := json.Marshal(req.Meta)
		if err != nil {
			return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to marshal meta: %s", err.Error()))
		}
		createKeyReq.Meta = util.Pointer(string(b))
	}
	if req.Ratelimit != nil {
		createKeyReq.Ratelimit = &keysv1.Ratelimit{
			Limit:          req.Ratelimit.Limit,
			RefillRate:     req.Ratelimit.RefillRate,
			RefillInterval: req.Ratelimit.RefillInterval,
		}
		switch req.Ratelimit.Type {
		case "fast":
			createKeyReq.Ratelimit.Type = keysv1.RatelimitType_RATELIMIT_TYPE_FAST
		case "consistent":
			createKeyReq.Ratelimit.Type = keysv1.RatelimitType_RATELIMIT_TYPE_CONSISTENT
		}
	}
	if req.Remaining > 0 {
		createKeyReq.Remaining = util.Pointer(req.Remaining)
	}
	if req.Expires > 0 {
		createKeyReq.Expires = util.Pointer(req.Expires)
	}

	s.logger.Info().Interface("req", createKeyReq).Msg("calling keyService.CreateKey")
	key, err := s.keyService.CreateKey(ctx, createKeyReq)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to create key: %s", err.Error()))
	}

	return c.JSON(CreateKeyResponse{
		Key:   key.Key,
		KeyId: key.KeyId,
	})
}
