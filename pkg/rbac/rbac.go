package rbac

import (
	"fmt"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// RBAC provides methods for evaluating permissions against requirements.
// It implements the core permission checking logic for the RBAC system.
type RBAC struct {
	// Contains filtered or unexported fields
}

// New creates a new RBAC instance for permission evaluation.
//
// Example:
//
//	rbac := rbac.New()
func New() *RBAC {
	return &RBAC{}
}

// EvaluationResult contains the outcome of a permission evaluation.
// It indicates whether the permissions are valid and provides a
// human-readable message explaining any failures.
type EvaluationResult struct {
	// Valid indicates whether the permission check passed
	Valid bool

	// Message contains details about why permissions were denied
	// This field is empty when Valid is true
	Message string
}

// EvaluatePermissions checks if the provided permissions satisfy the requirements
// specified in the permission query. It returns an EvaluationResult indicating
// whether the permissions are valid and, if not, why they failed.
//
// The permissions parameter should contain a list of permission strings in the
// format "resourceType.resourceID.action".
//
// Example:
//
//	userPermissions := []string{
//	    "api.api1.read_api",
//	    "api.api1.update_api",
//	}
//
//	query := rbac.And(
//	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.ReadAPI}),
//	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.UpdateAPI}),
//	)
//
//	result, err := rbac.EvaluatePermissions(query, userPermissions)
//	if err != nil {
//	    log.Fatalf("Error evaluating permissions: %v", err)
//	}
//
//	if result.Valid {
//	    // User has the required permissions
//	} else {
//	    // User lacks permissions
//	    fmt.Printf("Access denied: %s\n", result.Message)
//	}
func (r *RBAC) EvaluatePermissions(query PermissionQuery, permissions []string) (EvaluationResult, error) {
	return r.evaluateQueryV1(query, permissions)
}

// evaluateQueryV1 recursively evaluates a permission query tree against the
// granted permission strings, combining child results per the node operator.
func (r *RBAC) evaluateQueryV1(query PermissionQuery, permissions []string) (EvaluationResult, error) {
	// Handle simple permission check
	if query.Value != "" {
		if evaluateLeafPermission(query, permissions) {
			return EvaluationResult{Valid: true, Message: ""}, nil
		}
		return EvaluationResult{
			Valid:   false,
			Message: fmt.Sprintf("Missing permission: '%s'", query.Value),
		}, nil
	}

	// Handle AND operation
	if query.Operation == OperatorAnd {
		for _, child := range query.Children {
			result, err := r.evaluateQueryV1(child, permissions)
			if err != nil {
				return EvaluationResult{}, err
			}
			if !result.Valid {
				return result, nil
			}
		}
		return EvaluationResult{Valid: true, Message: ""}, nil
	}

	// Handle OR operation
	if query.Operation == OperatorOr {
		missingPerms := make([]string, 0)
		for _, child := range query.Children {
			result, err := r.evaluateQueryV1(child, permissions)
			if err != nil {
				return EvaluationResult{}, err
			}
			if result.Valid {
				return result, nil
			}
			missingPerms = append(missingPerms, fmt.Sprintf("'%v'", child))
		}

		return EvaluationResult{
			Valid:   false,
			Message: fmt.Sprintf("Missing one of these permissions: [%s]", strings.Join(missingPerms, ", ")),
		}, nil
	}

	return EvaluationResult{}, fault.New(
		fmt.Sprintf("query has invalid structure: operation=%s, value=%s, children=%d", query.Operation, query.Value, len(query.Children)),
		fault.Code(codes.App.Internal.UnexpectedError.URN()),
		fault.Public("The permission query has an invalid structure and cannot be evaluated."),
	)
}

// evaluateLeafPermission is the shared leaf evaluator for the boolean query
// tree. Exact string matching is always available. Unkey wildcard matching is
// an explicit opt-in carried by U(), not a behavior inferred from string shape.
func evaluateLeafPermission(query PermissionQuery, permissions []string) bool {
	if slices.Contains(permissions, query.Value) {
		return true
	}

	if !query.matchUnkeyPermission {
		return false
	}

	requiredPermission, err := parseUrnPermission(query.Value)
	if err != nil {
		return false
	}

	return evaluateUnkeyPermission(requiredPermission, permissions)
}

// ParseQuery parses a SQL-like permission query string and returns a PermissionQuery.
//
// Supported syntax:
//   - Permissions: alphanumeric characters, dots, underscores, hyphens, colons, asterisks, forward slashes (e.g., "api.key1.read_key", "system:admin", "api.*", "/api/v1/xxx")
//   - Operators: AND, OR (case-insensitive)
//   - Grouping: parentheses ()
//   - Precedence: AND has higher precedence than OR
//
// Important: Asterisks (*) in permission names are treated as literal characters,
// NOT as wildcard patterns. For example, "api.*" will only match a permission
// literally named "api.*", not permissions like "api.read" or "api.write".
//
// Examples:
//   - "api.key1.read_key"
//   - "api.*" (matches only the literal permission "api.*")
//   - "/api/v1/xxx" (path-like permission names)
//   - "perm1 AND perm2"
//   - "perm1 OR perm2 AND perm3" (parsed as "perm1 OR (perm2 AND perm3)")
//   - "(perm1 OR perm2) AND perm3"
//
// Limits:
//   - Maximum query length: 1000 characters
//   - Maximum permissions: 100
func ParseQuery(query string) (PermissionQuery, error) {
	return parseQuery(query)
}
