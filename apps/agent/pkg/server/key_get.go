package server

import (
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

type GetKeyRequest struct {
	KeyId string `validate:"required"`
}

type GetKeyResponse = keyResponse

func (s *Server) getKey(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.getKey")
	defer span.End()
	req := GetKeyRequest{
		KeyId: c.Params("keyId"),
	}

	err := s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}

	authHash, err := getKeyHash(c.Get("Authorization"))
	if err != nil {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, err.Error())
	}

	authKey, found, err := s.db.FindKeyByHash(ctx, authHash)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, fmt.Sprintf("unable to find key: %s", err.Error()))
	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("unable to find key by hash: %s", authHash))
	}

	if authKey.ForWorkspaceId == "" {
		return errors.NewHttpError(c, errors.BAD_REQUEST, "wrong key type")
	}

	key, found, err := s.db.FindKeyById(ctx, req.KeyId)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, err.Error())

	}
	if !found {
		return errors.NewHttpError(c, errors.NOT_FOUND, fmt.Sprintf("key %s not found", req.KeyId))
	}
	if key.WorkspaceId != authKey.ForWorkspaceId {
		return errors.NewHttpError(c, errors.UNAUTHORIZED, "workspace access denied")
	}

	api, found, err := s.db.FindApiByKeyAuthId(ctx, key.KeyAuthId)
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
