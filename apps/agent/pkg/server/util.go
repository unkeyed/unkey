package server

import (
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/apps/agent/pkg/hash"
)

// Return the hash of the key used for authentication
func getKeyHash(header string) (string, error) {
	if header == "" {
		return "", fmt.Errorf("authorization header is empty")
	}

	key := strings.TrimPrefix(header, "Bearer ")
	if key == "" {
		return "", fmt.Errorf("authorization header is malformed")
	}
	return hash.Sha256(key), nil

}
