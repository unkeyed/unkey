package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/errors"
)

type GetKeyStatsRequest struct {
	KeyId string `validate:"required"`
}

type usageRecord struct {
	Time  int64 `json:"time"`
	Value int64 `json:"value"`
}

type GetKeyStatsResponse struct {
	Usage []usageRecord `json:"usage"`
}

func (s *Server) getKeyStats(c *fiber.Ctx) error {
	ctx, span := s.tracer.Start(c.UserContext(), "server.getKey")
	defer span.End()
	req := GetKeyStatsRequest{
		KeyId: c.Params("keyId"),
	}

	err := s.validator.Struct(req)
	if err != nil {
		return errors.NewHttpError(c, errors.BAD_REQUEST, err.Error())
	}

	authorizedWorkspaceId, err := s.authorizeRootKey(ctx, c.Get(authorizationHeader))
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

	keyStats, err := s.analytics.GetKeyStats(ctx, key.Id)
	if err != nil {
		return errors.NewHttpError(c, errors.INTERNAL_SERVER_ERROR, "unable to load stats")
	}

	res := GetKeyStatsResponse{
		Usage: make([]usageRecord, len(keyStats.Usage)),
	}
	for i, day := range keyStats.Usage {
		res.Usage[i] = usageRecord{
			Time:  day.Time,
			Value: day.Value,
		}
	}

	return c.JSON(res)
}
