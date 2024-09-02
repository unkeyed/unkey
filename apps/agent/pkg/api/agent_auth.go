package api

import (
	"crypto/subtle"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/ftag"
	"github.com/gofiber/fiber/v2"
)

func (s *Server) BearerAuthFromSecret(secret string) fiber.Handler {

	secretB := []byte(secret)

	return func(c *fiber.Ctx) error {

		authorizationHeader := c.Get("Authorization")
		if authorizationHeader == "" {
			return fault.New("Authorization header is required", ftag.With(ftag.Unauthenticated))
		}

		token := strings.TrimPrefix(authorizationHeader, "Bearer ")
		if token == "" {
			return fault.New("Bearer token is required", ftag.With(ftag.Unauthenticated))
		}

		if subtle.ConstantTimeCompare([]byte(token), secretB) != 1 {

			return fault.New("Bearer token is invalid", ftag.With(ftag.Unauthenticated))
		}

		return c.Next()
	}

}
