package server

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
)

type GetKeyStatsRequest struct {
	KeyId string `validate:"required"`
}

type usageRecord struct {
	Time          int64 `json:"time"`
	Success       int64 `json:"success"`
	RateLimited   int64 `json:"rateLimited"`
	UsageExceeded int64 `json:"usageExceeded"`
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
		return newHttpError(c, BAD_REQUEST, err.Error())
	}

	auth, err := s.authorizeKey(ctx, c)
	if err != nil {
		return newHttpError(c, UNAUTHORIZED, err.Error())
	}
	if !auth.IsRootKey {
		return newHttpError(c, UNAUTHORIZED, "root key required")
	}
	key, found, err := cache.WithCache(s.keyCache, s.db.FindKeyById)(ctx, req.KeyId)
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, err.Error())
	}
	if !found {
		return newHttpError(c, NOT_FOUND, fmt.Sprintf("key %s not found", req.KeyId))
	}
	if key.WorkspaceId != auth.AuthorizedWorkspaceId {
		return newHttpError(c, UNAUTHORIZED, "workspace access denied")
	}
	api, found, err := cache.WithCache(s.apiCache, s.db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, err.Error())
	}
	if !found {
		return newHttpError(c, NOT_FOUND, fmt.Sprintf("api %s not found", key.KeyAuthId))
	}

	keyStats, err := s.analytics.GetKeyStats(ctx, key.WorkspaceId, api.ApiId, key.KeyId)
	if err != nil {
		return newHttpError(c, INTERNAL_SERVER_ERROR, "unable to load stats")
	}

	res := GetKeyStatsResponse{
		Usage: make([]usageRecord, len(keyStats.Usage)),
	}
	for i, day := range keyStats.Usage {
		res.Usage[i] = usageRecord{
			Time:          day.Time,
			Success:       day.Success,
			RateLimited:   day.RateLimited,
			UsageExceeded: day.UsageExceeded,
		}
	}

	return c.JSON(res)
}
