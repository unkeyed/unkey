// Package rbac implements a flexible Role-Based Access Control system for
// managing permissions and authorization checks.
//
// The package provides a way to define fine-grained permissions using a
// resource type, resource ID, and action tuple model, and to evaluate whether
// a set of permissions satisfies specific access requirements. This approach
// allows for both simple and complex authorization rules through boolean
// operations like AND and OR.
//
// The package supports two ways to construct permission queries:
// 1. Programmatic construction using helper functions
// 2. SQL-like query strings that are parsed at runtime
//
// # Key Concepts
//
//   - ResourceType: Categories of resources that can be protected (e.g., "api", "identity")
//   - ResourceID: Specific instance identifier within a resource type
//   - ActionType: Operations that can be performed on resources (e.g., "read", "write")
//   - Tuple: Combination of ResourceType, ResourceID, and ActionType that defines a permission
//   - PermissionQuery: Logical expressions for permission evaluation using AND/OR operations
//
// # Programmatic Query Construction
//
// Basic usage with programmatic query construction:
//
//	// Create an RBAC instance
//	rbac := rbac.New()
//
//	// Define permissions a user has
//	userPermissions := []string{
//	    "api:api1:read_api",
//	    "api:api1:update_api",
//	    "ratelimit:ns1:create_namespace",
//	}
//
//	// Create a permission query using helper functions
//	query := rbac.And(
//	    rbac.T(rbac.Tuple{
//	        ResourceType: rbac.Api,
//	        ResourceID:   "api1",
//	        Action:       rbac.ReadAPI,
//	    }),
//	    rbac.T(rbac.Tuple{
//	        ResourceType: rbac.Api,
//	        ResourceID:   "api1",
//	        Action:       rbac.UpdateAPI,
//	    }),
//	)
//
//	// Evaluate the permission query
//	result, err := rbac.EvaluatePermissions(query, userPermissions)
//	if err != nil {
//	    log.Fatalf("Error evaluating permissions: %v", err)
//	}
//
//	if result.Valid {
//	    // User has the required permissions
//	    performAction()
//	} else {
//	    // User lacks permissions
//	    fmt.Printf("Access denied: %s\n", result.Message)
//	}
//
// # SQL-like Query Parsing
//
// The package also supports parsing SQL-like permission query strings for
// dynamic authorization rules. This is useful when permission requirements
// are stored in configuration files or databases.
//
// SQL-like query usage:
//
//	// Parse a SQL-like permission query
//	query, err := rbac.ParseQuery("api.key1.read_key AND (api.key2.read_key OR api.key3.read_key)")
//	if err != nil {
//	    log.Fatalf("Failed to parse query: %v", err)
//	}
//
//	// Evaluate the parsed query
//	result, err := rbac.EvaluatePermissions(query, userPermissions)
//	if err != nil {
//	    log.Fatalf("Error evaluating permissions: %v", err)
//	}
//
// # SQL-like Query Syntax
//
// The SQL-like query parser supports:
//   - Permission identifiers: alphanumeric characters, dots, underscores, hyphens
//   - Logical operators: AND, OR (case-insensitive)
//   - Grouping: parentheses () for overriding precedence
//   - Precedence: AND has higher precedence than OR (like SQL)
//
// Examples:
//   - "api.key1.read_key"
//   - "perm1 AND perm2"
//   - "perm1 OR perm2 AND perm3" (parsed as "perm1 OR (perm2 AND perm3)")
//   - "(perm1 OR perm2) AND perm3"
//
// Query limitations:
//   - Maximum query length: 1000 characters
//   - Maximum permissions per query: 100
//
// The package supports complex permission queries through logical operators,
// allowing you to express requirements like "user must have permission X AND
// either permission Y OR permission Z".
package rbac
