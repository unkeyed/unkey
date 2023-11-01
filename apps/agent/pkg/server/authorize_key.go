package server

import (
	"context"

	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	authenticationv1 "github.com/unkeyed/unkey/apps/agent/gen/proto/authentication/v1"

	"github.com/unkeyed/unkey/apps/agent/pkg/util"
)

// authorizeKey is a utility to extract all required fields from a request and
// call the key service to authorize the request.
func (s *Server) authorizeKey(ctx context.Context, c *fiber.Ctx) (*authenticationv1.VerifyKeyResponse, error) {

	header := c.Get("Authorization")
	if header == "" {
		return nil, fmt.Errorf("authorization header is required")
	}

	key := strings.TrimPrefix(header, "Bearer ")
	if key == "" {
		return nil, fmt.Errorf("key should be in bearer format")
	}

	return s.keyService.VerifyKey(ctx, &authenticationv1.VerifyKeyRequest{
		Key:        key,
		SourceIp:   c.Get("Fly-Client-IP"),
		Region:     s.region,
		EdgeRegion: util.Pointer(c.Get("Fly-Region")),
		UserAgent:  util.Pointer(c.Get("User-Agent")),
		Resource:   util.Pointer(c.Path()),
	})
}
