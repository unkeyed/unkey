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
// Queries can be constructed using the And(), Or(), and T() helper functions.
type PermissionQuery struct {
	// Operation specifies the logical operator for this node
	Operation QueryOperator `json:"operation,omitempty"`

	// Value contains the permission string for leaf nodes (OperatorNil)
	Value string `json:"value,omitempty"`

	// Children contains sub-queries for non-leaf nodes (OperatorAnd/OperatorOr)
	Children []PermissionQuery `json:"children,omitempty"`
}

// And creates a permission query that requires all child queries to be satisfied.
// The resulting query will only evaluate to true if all child queries are true.
//
// Example:
//
//	// Require both read and update permissions
//	query := rbac.And(
//	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.ReadAPI}),
//	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.UpdateAPI}),
//	)
func And(queries ...PermissionQuery) PermissionQuery {
	return PermissionQuery{
		Operation: OperatorAnd,
		Value:     "",
		Children:  queries,
	}
}

// Or creates a permission query that requires at least one child query to be satisfied.
// The resulting query will evaluate to true if any child query is true.
//
// Example:
//
//	// Allow either create or delete permissions
//	query := rbac.Or(
//	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.CreateAPI}),
//	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.DeleteAPI}),
//	)
func Or(queries ...PermissionQuery) PermissionQuery {
	return PermissionQuery{
		Operation: OperatorOr,
		Value:     "",
		Children:  queries,
	}
}

// T creates a leaf permission query that checks for a specific permission tuple.
// This function is typically used as a building block for more complex
// permission queries using And() and Or().
//
// Example:
//
//	// Create a query for a single permission
//	query := rbac.T(rbac.Tuple{
//	    ResourceType: rbac.Api,
//	    ResourceID:   "api1",
//	    Action:       rbac.ReadAPI,
//	})
func T(tuple Tuple) PermissionQuery {
	return PermissionQuery{
		Operation: OperatorNil,
		Value:     tuple.String(),
		Children:  []PermissionQuery{},
	}
}

// S creates a leaf permission query that checks for a specific permission tuple.
// This function is typically used as a building block for more complex
// permission queries using And() and Or().
//
// Example:
//
//	// Create a query for a single permission
//	query := rbac.S("resourceType.resourceID.action")
func S(s string) PermissionQuery {
	return PermissionQuery{
		Operation: OperatorNil,
		Value:     s,
		Children:  []PermissionQuery{},
	}
}
