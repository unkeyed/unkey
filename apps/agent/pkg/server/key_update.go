package server

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/entities"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
	"github.com/unkeyed/unkey/apps/agent/pkg/events"
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
		Limit          int32  `json:"limit" validate:"required"`
		RefillRate     int32  `json:"refillRate" validate:"required"`
		RefillInterval int32  `json:"refillInterval" validate:"required"`
	}] `json:"ratelimit"`
	Remaining nullish[int32] `json:"remaining"`
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
		return errors.NewHttpError(c, errors.BAD_REQUEST, fmt.Sprintf("unable to parse body: %s", err.Error()))
	}

	err = c.BodyParser(&req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, fmt.Sprintf("unable to parse body: %s", err.Error()))
	}

	err = s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, fmt.Sprintf("unable to validate body: %s", err.Error()))
	}

	if req.Expires.Defined && req.Expires.Value != nil && *req.Expires.Value > 0 && *req.Expires.Value < time.Now().UnixMilli() {
		return errors.NewHttpError(c, errors.BAD_REQUEST, "'expires' must be in the future, did you pass in a timestamp in seconds instead of milliseconds?")
	}

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c.Get(authorizationHeader))
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}
	// This is not cached on purpose
	// In case you're updating the same key in rapid succession or your requests are handled by
	// different machines, it was possible to overwrite the key with cached data
	key, found, err := s.db.FindKeyById(ctx, req.KeyId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find key: %s", err.Error()))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("key %s does not exist", req.KeyId))
	}
	if key.WorkspaceId != authorizedWorkspaceId {
		return errors.NewHttpError(c, errors.FORBIDDEN, "access to workspace denied")
	}

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
			key.Remaining = req.Remaining.Value
		} else {
			key.Remaining = nil
		}
	}

	err = s.db.UpdateKey(ctx, key)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to write key: %s", err.Error()))
	}
	s.keyCache.Set(ctx, key.Hash, key)

	s.events.EmitKeyEvent(ctx, events.KeyEvent{
		Type: events.KeyUpdated,
		Key: events.Key{
			Id:   key.Id,
			Hash: key.Hash,
		},
	})

	return c.JSON(UpdateKeyResponse{})
}
