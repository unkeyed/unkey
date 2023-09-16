package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/unkeyed/unkey/apps/agent/pkg/cache"
	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
)

func (s *Server) authorizeRootKey(ctx context.Context, header string) (authorizedWorkspace string, err error) {
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
	if !key.Expires.IsZero() && key.Expires.Before(time.Now()) {
		s.keyCache.Remove(ctx, h)
		err := s.db.DeleteKey(ctx, key.Id)
		if err != nil {
			return "", err
		}

		return "", fmt.Errorf("key not found")

	}

	if key.ForWorkspaceId == "" {
		return "", fmt.Errorf("wrong key")
	}

	return key.ForWorkspaceId, nil

}
