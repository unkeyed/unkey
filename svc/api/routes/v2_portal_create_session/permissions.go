package handler

import "github.com/unkeyed/unkey/pkg/rbac"

// readAnalyticsPermissions is the api-scoped legacy-tuple requirement for
// reading verification analytics for a keyspace.
//
// Unlike the key requirements this route borrows from the operator routes that
// enforce them, this one is defined locally: v2_analytics_get_verifications has
// no query object to share. It authorizes an api.*.read_analytics wildcard tuple
// and otherwise string-parses the principal's permissions to build ClickHouse
// row filters, so there is nothing to import. This mirrors what that route
// matches today, the api wildcard or the specific api.
//
// Do not add a keyspace log URN arm here yet. The analytics endpoint evaluates
// legacy tuples only. A keyspace log grant would otherwise let a caller mint a
// capability it could not exercise.
func readAnalyticsPermissions(apiID string) rbac.PermissionQuery {
	return rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadAnalytics,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   apiID,
			Action:       rbac.ReadAnalytics,
		}),
	)
}
