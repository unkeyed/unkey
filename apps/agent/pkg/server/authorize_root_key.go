package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/unkeyed/unkey/apps/agent/pkg/analytics"
	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
)

func (s *Server) authorizeRootKey(ctx context.Context, c *fiber.Ctx) (authorizedWorkspace string, err error) {
	header := c.Get("Authorization")
	if header == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	token := strings.TrimPrefix(header, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("authorization header is malformed")
	}

	h := hash.Sha256(token)

	key, found, err := cache.WithCache(s.keyCache, s.db.FindKeyByHash)(ctx, h)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("key not found")
	}
	api, found, err := cache.WithCache(s.apiCache, s.db.FindApiByKeyAuthId)(ctx, key.KeyAuthId)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("keyauth %s not found", key.KeyAuthId)
	}

	defer func() {
		s.analytics.PublishKeyVerificationEvent(ctx, analytics.KeyVerificationEvent{
			WorkspaceId:       key.WorkspaceId,
			ApiId:             api.Id,
			KeyId:             key.Id,
			Denied:            "",
			Time:              time.Now().UnixMilli(),
			Region:            s.region,
			EdgeRegion:        c.Get("Fly-Region"),
			UserAgent:         c.Get("User-Agent"),
			IpAddress:         c.Get("Fly-Client-IP"),
			RequestedResource: c.Path(),
		})
	}()

	if key.Expires != nil && time.UnixMilli(key.GetExpires()).Before(time.Now()) {
		s.keyCache.Remove(ctx, h)
		err := s.db.SoftDeleteKey(ctx, key.Id)
		if err != nil {
			return "", err
		}

		return "", fmt.Errorf("key not found")

	}

	if key.ForWorkspaceId == nil {
		return "", fmt.Errorf("wrong key")
	}

	return key.GetForWorkspaceId(), nil

}
