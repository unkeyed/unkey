//go:build !linux
// +build !linux

package metald

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

// initializeFirecrackerBackend returns an error on non-Linux platforms
func initializeFirecrackerBackend(ctx context.Context, cfg *Config, logger *slog.Logger, tlsProvider tlspkg.Provider) (types.Backend, error) {
	return nil, fmt.Errorf("firecracker backend is only supported on Linux")
}
