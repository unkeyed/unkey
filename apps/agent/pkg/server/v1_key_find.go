package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

type GetKeyRequestV1 struct {
	KeyId string `validate:"required"`
}

type GetKeyResponseV1 = keyResponse

func (s *Server) v1FindKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.v1FindKey")
	defer span.End()
	req := GetKeyRequest{
		KeyId: c.Query("keyId"),
	}

	err := s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c)
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}

	key, found, err := cache.WithCache(s.keyCache, s.db.FindKeyById)(ctx, req.KeyId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("key %s not found", req.KeyId))
	}
	if key.WorkspaceId != authorizedWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "workspace access denied")
	}
	api, found, err := cache.WithCache(s.apiCache, s.db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find api: %s", err.Error()))
	}
	if !found {

		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find api: %s", err.Error()))
	}

	res := GetKeyResponse{
		Id:             key.Id,
		ApiId:          api.Id,
		WorkspaceId:    key.WorkspaceId,
		Name:           key.Name,
		Start:          key.Start,
		OwnerId:        key.OwnerId,
		Meta:           key.Meta,
		CreatedAt:      key.CreatedAt.UnixMilli(),
		ForWorkspaceId: key.ForWorkspaceId,
	}
	if !key.Expires.IsZero() {
		res.Expires = key.Expires.UnixMilli()
	}
	if key.Ratelimit != nil {
		res.Ratelimit = &ratelimitSettng{
			Type:           key.Ratelimit.Type,
			Limit:          key.Ratelimit.Limit,
			RefillRate:     key.Ratelimit.RefillRate,
			RefillInterval: key.Ratelimit.RefillInterval,
		}
	}
	if key.Remaining != nil {
		res.Remaining = key.Remaining
	}

	return c.JSON(res)
}
