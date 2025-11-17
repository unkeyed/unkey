package deployments

import (
	"context"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
)

// Service handles deployment lookups
type Service interface {
	// LookupByHostname finds where to route a request based on hostname
	// Returns:
	//   - deployment, true, nil if found
	//   - nil, false, nil if not found
	//   - nil, false, error if lookup failed
	LookupByHostname(ctx context.Context, hostname string) (*partitionv1.Deployment, bool, error)
}
