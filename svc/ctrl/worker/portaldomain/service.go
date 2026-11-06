package portaldomain

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/domainverify"
)

// Service implements hydrav1.PortalDomainServiceServer, a virtual object keyed
// by portal domain id so one verification runs per row at a time.
type Service struct {
	hydrav1.UnimplementedPortalDomainServiceServer
	db            db.Database
	resolver      domainverify.Resolver
	environmentID string
}

var _ hydrav1.PortalDomainServiceServer = (*Service)(nil)

// Config holds configuration for creating a [Service].
type Config struct {
	DB db.Database

	// Resolver answers the DNS lookups verification makes. Production passes
	// [domainverify.SystemResolver].
	Resolver domainverify.Resolver

	// EnvironmentID is the portal app's production environment, which every
	// verified portal domain routes to. Empty leaves verified domains unrouted
	// and their workflows retrying until it is set.
	EnvironmentID string
}

// New creates a [Service] with the given configuration.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedPortalDomainServiceServer: hydrav1.UnimplementedPortalDomainServiceServer{},
		db:                                     cfg.DB,
		resolver:                               cfg.Resolver,
		environmentID:                          cfg.EnvironmentID,
	}
}
