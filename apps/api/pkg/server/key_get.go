package server

import (
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/api/pkg/database"
	"net/http"
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
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:  BAD_REQUEST,
			Error: fmt.Sprintf("unable to validate request: %s", err.Error()),
		})
	}

	authHash, err := getKeyHash(c.Get("Authorization"))
	if err != nil {
		return err
	}

	authKey, err := s.db.GetKeyByHash(ctx, authHash)
	if err != nil {
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
			return c.Status(http.StatusNotFound).JSON(ErrorResponse{
				Code:  NOT_FOUND,
				Error: fmt.Sprintf("unable to find key: %s", req.KeyId),
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

	api, err := s.db.GetApiByKeyAuthId(ctx, key.KeyAuthId)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:  INTERNAL_SERVER_ERROR,
			Error: fmt.Sprintf("unable to find api: %s", err.Error()),
		})
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
	if key.Remaining.Enabled {
		res.Remaining = &key.Remaining.Remaining
	}

	return c.JSON(res)
}
