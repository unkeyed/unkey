// Package flags reports database-backed boolean feature flags for a workspace.
// It owns its queries and cache, so callers ask for a flag by name and get a
// bool. A workspace override wins over the database default, including an
// explicit false; a missing definition uses the code default.
//
// # Adding a flag
//
// Each flag is one method on [Service], implemented with its slug and code
// default:
//
//	// In the Service interface:
//	IsPrivateNetworkingEnabled(ctx context.Context, workspaceID string) bool
//
//	// On the implementation:
//	func (s *service) IsPrivateNetworkingEnabled(ctx context.Context, workspaceID string) bool {
//		return s.enabled(ctx, workspaceID, "private_network_enable", false)
//	}
//
// Call sites then cannot misspell a slug or disagree on a default:
//
//	if flagService.IsPrivateNetworkingEnabled(ctx, workspaceID) {
//		return deployWithPrivateNetworking(ctx)
//	}
//
// Definitions and overrides are managed outside this package, and evaluation
// never writes to the database. A method whose definition does not exist yet
// returns its code default, so code can ship before the flag row.
//
// Reads use the read replica and a process-local cache, so changes are
// eventually consistent. Cached values expire after one minute; a failed load
// uses the code default, so choose it deliberately. [NewNoop] always uses code
// defaults. This is not an authorization mechanism.
package flags
