// Package rbac evaluates platform and customer-defined permission queries.
//
// Platform authorization uses [U] with typed resources from
// [github.com/unkeyed/unkey/pkg/urn] and actions from
// [github.com/unkeyed/unkey/pkg/rbac/permissions]. [S] checks an arbitrary
// customer-defined permission exactly, while [ParseQuery] parses boolean
// expressions containing those customer permissions.
//
// Platform permission example:
//
//	query := rbac.U(
//	    urn.New().Workspace("ws_123").Project("proj_123"),
//	    permissions.Read,
//	)
//
// Customer permission example:
//
//	query, err := rbac.ParseQuery("documents.read AND (documents.write OR billing.admin)")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := rbac.EvaluatePermissions(query, []string{"documents.read", "documents.write"})
package rbac
