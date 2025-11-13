package routing

import (
	"context"
	"net/url"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
)

// Service handles gateway configuration lookup and VM selection.
type Service interface {
	// GetConfig finds gateway configuration and workspace ID based on the request host
	GetConfig(ctx context.Context, host string) (*ConfigWithWorkspace, error)

	// SelectVM picks an available VM from the gateway's VM list
	SelectVM(ctx context.Context, config *partitionv1.GatewayConfig) (*url.URL, error)
}

// Config holds configuration for the routing service.
type Config struct {
	DB     db.Database
	Logger logging.Logger
	Clock  clock.Clock

	GatewayConfigCache cache.Cache[string, ConfigWithWorkspace]
	InstanceCache      cache.Cache[string, pdb.Instance]
}
