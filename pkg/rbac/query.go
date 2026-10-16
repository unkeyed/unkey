// query.go
package rbac

// QueryOperator defines the logical operators used in permission queries.
type QueryOperator string

// Predefined query operators for building permission expressions.
const (
	// OperatorNil represents a leaf node in the permission query tree,
	// containing a direct permission value.
	OperatorNil QueryOperator = ""

	// OperatorAnd requires all child queries to be satisfied.
	OperatorAnd QueryOperator = "and"

	// OperatorOr requires at least one child query to be satisfied.
	OperatorOr QueryOperator = "or"
)

// PermissionQuery represents a logical expression for evaluating permissions.
// It can be a simple permission check or a complex boolean expression using
// AND/OR operators with nested conditions.
//
// Queries can be constructed using [And], [Or], [S], and [U].
type PermissionQuery struct {
	// Operation specifies the logical operator for this node
	Operation QueryOperator `json:"operation,omitempty"`

	// Value contains the permission string for leaf nodes (OperatorNil)
	Value string `json:"value,omitempty"`

	// Children contains sub-queries for non-leaf nodes (OperatorAnd/OperatorOr)
	Children []PermissionQuery `json:"children,omitempty"`

	// When true, this leaf opts into Unkey resource permission matching. The
	// marker is intentionally not serialized or set by ParseQuery because
	// wildcard semantics must be chosen by typed call sites through U().
	matchUnkeyPermission bool
}

// And creates a permission query that requires all child queries to be satisfied.
// The resulting query will only evaluate to true if all child queries are true.
//
// Example:
//
//	// Require both customer-defined permissions.
//	query := rbac.And(
//	    rbac.S("documents.read"),
//	    rbac.S("documents.write"),
//	)
func And(queries ...PermissionQuery) PermissionQuery {
	return PermissionQuery{
		Operation:            OperatorAnd,
		Value:                "",
		Children:             queries,
		matchUnkeyPermission: false,
	}
}

// Or creates a permission query that requires at least one child query to be satisfied.
// The resulting query will evaluate to true if any child query is true.
//
// Example:
//
//	// Allow either customer-defined permission.
//	query := rbac.Or(
//	    rbac.S("documents.read"),
//	    rbac.S("documents.write"),
//	)
func Or(queries ...PermissionQuery) PermissionQuery {
	return PermissionQuery{
		Operation:            OperatorOr,
		Value:                "",
		Children:             queries,
		matchUnkeyPermission: false,
	}
}

// S creates a leaf query for an exact customer-defined permission string.
// This function is typically used as a building block for more complex
// permission queries using And() and Or().
//
// Example:
//
//	// Create a query for a single permission
//	query := rbac.S("documents.read")
func S(s string) PermissionQuery {
	return PermissionQuery{
		Operation:            OperatorNil,
		Value:                s,
		Children:             []PermissionQuery{},
		matchUnkeyPermission: false,
	}
}
