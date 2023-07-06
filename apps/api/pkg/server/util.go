package server

import (
	"strings"

	"github.com/chronark/unkey/apps/api/pkg/hash"
	"github.com/gofiber/fiber/v2"
)

// Return the hash of the key used for authentication
func getKeyHash(header string) (string, error) {
	if header == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized)
	}

	key := strings.TrimPrefix(header, "Bearer ")
	if key == "" {
		return "", fiber.NewError(fiber.StatusUnauthorized)
	}
	return hash.Sha256(key), nil

}
