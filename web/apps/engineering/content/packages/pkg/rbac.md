---
title: rbac
description: "implements a flexible Role-Based Access Control system for"
---

Package rbac implements a flexible Role-Based Access Control system for managing permissions and authorization checks.

The package provides a way to define fine-grained permissions using a resource type, resource ID, and action tuple model, and to evaluate whether a set of permissions satisfies specific access requirements. This approach allows for both simple and complex authorization rules through boolean operations like AND and OR.

The package supports two ways to construct permission queries: 1. Programmatic construction using helper functions 2. SQL-like query strings that are parsed at runtime

### Key Concepts

  - ResourceType: Categories of resources that can be protected (e.g., "api", "identity")
  - ResourceID: Specific instance identifier within a resource type
  - ActionType: Operations that can be performed on resources (e.g., "read", "write")
  - Tuple: Combination of ResourceType, ResourceID, and ActionType that defines a permission
  - PermissionQuery: Logical expressions for permission evaluation using AND/OR operations

### Programmatic Query Construction

Basic usage with programmatic query construction:

	// Create an RBAC instance
	rbac := rbac.New()

	// Define permissions a user has
	userPermissions := []string{
	    "api:api1:read_api",
	    "api:api1:update_api",
	    "ratelimit:ns1:create_namespace",
	}

	// Create a permission query using helper functions
	query := rbac.And(
	    rbac.T(rbac.Tuple{
	        ResourceType: rbac.Api,
	        ResourceID:   "api1",
	        Action:       rbac.ReadAPI,
	    }),
	    rbac.T(rbac.Tuple{
	        ResourceType: rbac.Api,
	        ResourceID:   "api1",
	        Action:       rbac.UpdateAPI,
	    }),
	)

	// Evaluate the permission query
	result, err := rbac.EvaluatePermissions(query, userPermissions)
	if err != nil {
	    log.Fatalf("Error evaluating permissions: %v", err)
	}

	if result.Valid {
	    // User has the required permissions
	    performAction()
	} else {
	    // User lacks permissions
	    fmt.Printf("Access denied: %s\n", result.Message)
	}

### SQL-like Query Parsing

The package also supports parsing SQL-like permission query strings for dynamic authorization rules. This is useful when permission requirements are stored in configuration files or databases.

SQL-like query usage:

	// Parse a SQL-like permission query
	query, err := rbac.ParseQuery("api.key1.read_key AND (api.key2.read_key OR api.key3.read_key)")
	if err != nil {
	    log.Fatalf("Failed to parse query: %v", err)
	}

	// Evaluate the parsed query
	result, err := rbac.EvaluatePermissions(query, userPermissions)
	if err != nil {
	    log.Fatalf("Error evaluating permissions: %v", err)
	}

### SQL-like Query Syntax

The SQL-like query parser supports:

  - Permission identifiers: alphanumeric characters, dots, underscores, hyphens
  - Logical operators: AND, OR (case-insensitive)
  - Grouping: parentheses () for overriding precedence
  - Precedence: AND has higher precedence than OR (like SQL)

Examples:

  - "api.key1.read\_key"
  - "perm1 AND perm2"
  - "perm1 OR perm2 AND perm3" (parsed as "perm1 OR (perm2 AND perm3)")
  - "(perm1 OR perm2) AND perm3"

Query limitations:

  - Maximum query length: 1000 characters
  - Maximum permissions per query: 100

The package supports complex permission queries through logical operators, allowing you to express requirements like "user must have permission X AND either permission Y OR permission Z".

query.go

## Functions


## Types

### type ActionType

```go
type ActionType string
```

ActionType represents operations that can be performed on resources. Each resource type can have its own set of applicable actions.

Predefined API actions. These constants define operations that can be performed on API resources.
```go
const (
	// ReadAPI permits viewing API details
	ReadAPI ActionType = "read_api"

	// CreateAPI permits creating new APIs
	CreateAPI ActionType = "create_api"

	// DeleteAPI permits removing existing APIs
	DeleteAPI ActionType = "delete_api"

	// UpdateAPI permits modifying API configurations
	UpdateAPI ActionType = "update_api"

	// CreateKey permits generating new API keys
	CreateKey ActionType = "create_key"

	// UpdateKey permits modifying existing API keys
	UpdateKey ActionType = "update_key"

	// DeleteKey permits removing API keys
	DeleteKey ActionType = "delete_key"

	// EncryptKey permits encrypting API keys
	EncryptKey ActionType = "encrypt_key"

	// DecryptKey permits decrypting API keys
	DecryptKey ActionType = "decrypt_key"

	// ReadKey permits viewing API key details
	ReadKey ActionType = "read_key"

	// VerifyKey permits verifying API keys
	VerifyKey ActionType = "verify_key"

	// ReadAnalytics permits viewing API analytics
	ReadAnalytics ActionType = "read_analytics"
)
```

Predefined rate limiting actions. These constants define operations that can be performed on rate limiting resources.
```go
const (
	// Limit permits applying rate limits
	Limit ActionType = "limit"

	// CreateNamespace permits creating rate limit namespaces
	CreateNamespace ActionType = "create_namespace"

	// ReadNamespace permits viewing rate limit namespace details
	ReadNamespace ActionType = "read_namespace"

	// UpdateNamespace permits modifying rate limit namespaces
	UpdateNamespace ActionType = "update_namespace"

	// DeleteNamespace permits removing rate limit namespaces
	DeleteNamespace ActionType = "delete_namespace"

	// SetOverride permits setting rate limit overrides
	SetOverride ActionType = "set_override"

	// ReadOverride permits viewing rate limit override details
	ReadOverride ActionType = "read_override"

	// DeleteOverride permits removing rate limit overrides
	DeleteOverride ActionType = "delete_override"

	// ListOverrides permits viewing rate limit override lists
	ListOverrides ActionType = "list_overrides"
)
```

Predefined RBAC actions. These constants define operations that can be performed on permission and role resources.
```go
const (
	// CreatePermission permits defining new permissions
	CreatePermission ActionType = "create_permission"

	// UpdatePermission permits modifying existing permissions
	UpdatePermission ActionType = "update_permission"

	// DeletePermission permits removing permissions
	DeletePermission ActionType = "delete_permission"

	// ReadPermission permits viewing permission details
	ReadPermission ActionType = "read_permission"

	// CreateRole permits defining new roles
	CreateRole ActionType = "create_role"

	// UpdateRole permits modifying existing roles
	UpdateRole ActionType = "update_role"

	// DeleteRole permits removing roles
	DeleteRole ActionType = "delete_role"

	// ReadRole permits viewing role details
	ReadRole ActionType = "read_role"

	// AddPermissionToKey permits assigning permissions directly to keys
	AddPermissionToKey ActionType = "add_permission_to_key"

	// RemovePermissionFromKey permits unassigning permissions from keys
	RemovePermissionFromKey ActionType = "remove_permission_from_key"

	// AddRoleToKey permits assigning roles to keys
	AddRoleToKey ActionType = "add_role_to_key"

	// RemoveRoleFromKey permits unassigning roles from keys
	RemoveRoleFromKey ActionType = "remove_role_from_key"

	// AddPermissionToRole permits assigning permissions to roles
	AddPermissionToRole ActionType = "add_permission_to_role"

	// RemovePermissionFromRole permits unassigning permissions from roles
	RemovePermissionFromRole ActionType = "remove_permission_from_role"
)
```

Predefined identity actions. These constants define operations that can be performed on identity resources.
```go
const (
	// CreateIdentity permits creating new identities
	CreateIdentity ActionType = "create_identity"

	// ReadIdentity permits viewing identity details
	ReadIdentity ActionType = "read_identity"

	// UpdateIdentity permits modifying existing identities
	UpdateIdentity ActionType = "update_identity"

	// DeleteIdentity permits removing identities
	DeleteIdentity ActionType = "delete_identity"
)
```

Predefined project actions. These constants define operations that can be performed on project deployment resources.
```go
const (
	// CreateDeployment permits creating new deployments
	CreateDeployment ActionType = "create_deployment"

	// ReadDeployment permits viewing deployment details
	ReadDeployment ActionType = "read_deployment"

	// GenerateUploadURL permits generating S3 upload URLs for build contexts
	GenerateUploadURL ActionType = "generate_upload_url"
)
```

### type EvaluationResult

```go
type EvaluationResult struct {
	// Valid indicates whether the permission check passed
	Valid bool

	// Message contains details about why permissions were denied
	// This field is empty when Valid is true
	Message string
}
```

EvaluationResult contains the outcome of a permission evaluation. It indicates whether the permissions are valid and provides a human-readable message explaining any failures.

### type PermissionQuery

```go
type PermissionQuery struct {
	// Operation specifies the logical operator for this node
	Operation QueryOperator `json:"operation,omitempty"`

	// Value contains the permission string for leaf nodes (OperatorNil)
	Value string `json:"value,omitempty"`

	// Children contains sub-queries for non-leaf nodes (OperatorAnd/OperatorOr)
	Children []PermissionQuery `json:"children,omitempty"`
}
```

PermissionQuery represents a logical expression for evaluating permissions. It can be a simple permission check or a complex boolean expression using AND/OR operators with nested conditions.

Queries can be constructed using the And(), Or(), and T() helper functions.

#### func And

```go
func And(queries ...PermissionQuery) PermissionQuery
```

And creates a permission query that requires all child queries to be satisfied. The resulting query will only evaluate to true if all child queries are true.

Example:

	// Require both read and update permissions
	query := rbac.And(
	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.ReadAPI}),
	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.UpdateAPI}),
	)

#### func Or

```go
func Or(queries ...PermissionQuery) PermissionQuery
```

Or creates a permission query that requires at least one child query to be satisfied. The resulting query will evaluate to true if any child query is true.

Example:

	// Allow either create or delete permissions
	query := rbac.Or(
	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.CreateAPI}),
	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.DeleteAPI}),
	)

#### func ParseQuery

```go
func ParseQuery(query string) (PermissionQuery, error)
```

ParseQuery parses a SQL-like permission query string and returns a PermissionQuery.

Supported syntax:

  - Permissions: alphanumeric characters, dots, underscores, hyphens, colons, asterisks, forward slashes (e.g., "api.key1.read\_key", "system:admin", "api.\*", "/api/v1/xxx")
  - Operators: AND, OR (case-insensitive)
  - Grouping: parentheses ()
  - Precedence: AND has higher precedence than OR

Important: Asterisks (\*) in permission names are treated as literal characters, NOT as wildcard patterns. For example, "api.\*" will only match a permission literally named "api.\*", not permissions like "api.read" or "api.write".

Examples:

  - "api.key1.read\_key"
  - "api.\*" (matches only the literal permission "api.\*")
  - "/api/v1/xxx" (path-like permission names)
  - "perm1 AND perm2"
  - "perm1 OR perm2 AND perm3" (parsed as "perm1 OR (perm2 AND perm3)")
  - "(perm1 OR perm2) AND perm3"

Limits:

  - Maximum query length: 1000 characters
  - Maximum permissions: 100

#### func S

```go
func S(s string) PermissionQuery
```

S creates a leaf permission query that checks for a specific permission tuple. This function is typically used as a building block for more complex permission queries using And() and Or().

Example:

	// Create a query for a single permission
	query := rbac.S("resourceType.resourceID.action")

#### func T

```go
func T(tuple Tuple) PermissionQuery
```

T creates a leaf permission query that checks for a specific permission tuple. This function is typically used as a building block for more complex permission queries using And() and Or().

Example:

	// Create a query for a single permission
	query := rbac.T(rbac.Tuple{
	    ResourceType: rbac.Api,
	    ResourceID:   "api1",
	    Action:       rbac.ReadAPI,
	})

### type QueryOperator

```go
type QueryOperator string
```

QueryOperator defines the logical operators used in permission queries.

Predefined query operators for building permission expressions.
```go
const (
	// OperatorNil represents a leaf node in the permission query tree,
	// containing a direct permission value.
	OperatorNil QueryOperator = ""

	// OperatorAnd requires all child queries to be satisfied.
	OperatorAnd QueryOperator = "and"

	// OperatorOr requires at least one child query to be satisfied.
	OperatorOr QueryOperator = "or"
)
```

### type RBAC

```go
type RBAC struct {
}
```

RBAC provides methods for evaluating permissions against requirements. It implements the core permission checking logic for the RBAC system.

#### func New

```go
func New() *RBAC
```

New creates a new RBAC instance for permission evaluation.

Example:

	rbac := rbac.New()

#### func (RBAC) EvaluatePermissions

```go
func (r *RBAC) EvaluatePermissions(query PermissionQuery, permissions []string) (EvaluationResult, error)
```

EvaluatePermissions checks if the provided permissions satisfy the requirements specified in the permission query. It returns an EvaluationResult indicating whether the permissions are valid and, if not, why they failed.

The permissions parameter should contain a list of permission strings in the format "resourceType.resourceID.action".

Example:

	userPermissions := []string{
	    "api.api1.read_api",
	    "api.api1.update_api",
	}

	query := rbac.And(
	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.ReadAPI}),
	    rbac.T(rbac.Tuple{ResourceType: rbac.Api, ResourceID: "api1", Action: rbac.UpdateAPI}),
	)

	result, err := rbac.EvaluatePermissions(query, userPermissions)
	if err != nil {
	    log.Fatalf("Error evaluating permissions: %v", err)
	}

	if result.Valid {
	    // User has the required permissions
	} else {
	    // User lacks permissions
	    fmt.Printf("Access denied: %s\n", result.Message)
	}

### type ResourceType

```go
type ResourceType string
```

ResourceType represents categories of resources that can be protected by the RBAC system. It helps organize permissions by functional area.

Predefined resource types. These constants provide a standardized set of resource categories used throughout the system.
```go
const (
	// Api represents API-related resources, such as endpoints, keys, etc.
	Api ResourceType = "api"

	// Ratelimit represents rate limiting resources and configuration
	Ratelimit ResourceType = "ratelimit"

	// Rbac represents permissions and roles management resources
	Rbac ResourceType = "rbac"

	// Identity represents user and identity management resources
	Identity ResourceType = "identity"

	// Deploy represents deployment resources and operations
	Project ResourceType = "project"
)
```

### type Tuple

```go
type Tuple struct {
	// ResourceType defines the category of resource being accessed
	ResourceType ResourceType

	// ResourceID identifies the specific resource instance
	ResourceID string

	// Action specifies the operation being performed on the resource
	Action ActionType
}
```

Tuple represents a specific permission as a combination of resource type, resource ID, and action. It forms the basic unit of permission definition in the RBAC system.

Example:

	// Permission to read a specific API
	readApiPermission := rbac.Tuple{
	    ResourceType: rbac.Api,
	    ResourceID:   "api1",
	    Action:       rbac.ReadAPI,
	}

#### func TupleFromString

```go
func TupleFromString(s string) (Tuple, error)
```

TupleFromString parses a string in the format "resourceType.resourceID.action" into a Tuple. Returns an error if the string format is invalid.

Example:

	tuple, err := rbac.TupleFromString("api.api1.read_api")
	if err != nil {
	    log.Fatalf("Invalid permission format: %v", err)
	}

#### func (Tuple) String

```go
func (t Tuple) String() string
```

String converts a Tuple to its string representation in the format "resourceType.resourceID.action" (e.g., "api.api1.read\_api").

This string format is used when storing and comparing permissions.

