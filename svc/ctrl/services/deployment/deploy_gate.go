package deployment

import "database/sql"

// deployEntitled reports whether a workspace may create Unkey Deploy
// deployments. A workspace is entitled if it has a synced Deploy plan
// (deploy_plan, mirrored from Stripe) or a manual override
// (deploy_plan_override, for internal/comped workspaces). NULL or empty for
// both means no entitlement. Mirrors the identical predicate in the project
// service; both gate the same entitlement at their own creation boundary.
func deployEntitled(plan, override sql.NullString) bool {
	return isSet(plan) || isSet(override)
}

func isSet(v sql.NullString) bool {
	return v.Valid && v.String != ""
}
