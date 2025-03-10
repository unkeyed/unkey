// Package rbac implements a flexible Role-Based Access Control system for
// managing permissions and authorization checks.
//
// The package provides a way to define fine-grained permissions using a
// resource type, resource ID, and action tuple model, and to evaluate whether
// a set of permissions satisfies specific access requirements. This approach
// allows for both simple and complex authorization rules through boolean
// operations like AND and OR.
//
// Key concepts:
//
//   - ResourceType: Categories of resources that can be protected (e.g., "api", "identity")
//   - ResourceID: Specific instance identifier within a resource type
//   - ActionType: Operations that can be performed on resources (e.g., "read", "write")
//   - Tuple: Combination of ResourceType, ResourceID, and ActionType that defines a permission
//   - PermissionQuery: Logical expressions for permission evaluation using AND/OR operations
//
// Basic usage:
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
//	// Create a permission query
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
// The package supports complex permission queries through logical operators,
// allowing you to express requirements like "user must have permission X AND
// either permission Y OR permission Z".
package rbac
