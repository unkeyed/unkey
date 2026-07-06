package billing

import (
	"context"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// LoadEnabledResources returns the keyspace ids and ratelimit namespace ids a
// workspace has enabled for end-user billing, read from
// billing_billable_resources. Enablement is opt-in: both slices are empty when
// the workspace has enabled nothing, in which case no usage is billable. The
// same loader backs the period-close push and the invoice-draft preview so both
// scope usage identically (R19 parity).
func LoadEnabledResources(ctx context.Context, database db.Database, workspaceID string) (keyspaces, namespaces []string, err error) {
	rows, err := db.Query.ListBillingBillableResources(ctx, database.RO(), workspaceID)
	if err != nil {
		return nil, nil, fault.Wrap(err, fault.Internal("failed to list billable resources"))
	}
	for _, r := range rows {
		switch r.ResourceType {
		case db.BillingBillableResourcesResourceTypeKeyspace:
			keyspaces = append(keyspaces, r.ResourceID)
		case db.BillingBillableResourcesResourceTypeNamespace:
			namespaces = append(namespaces, r.ResourceID)
		}
	}
	return keyspaces, namespaces, nil
}

// LoadEnabledResources returns the workspace's billing-enabled keyspace and
// namespace ids using the resolver's database. Convenience for API handlers
// that already hold a *Resolver and would otherwise need a separate DB handle.
func (r *Resolver) LoadEnabledResources(ctx context.Context, workspaceID string) (keyspaces, namespaces []string, err error) {
	return LoadEnabledResources(ctx, r.database, workspaceID)
}
