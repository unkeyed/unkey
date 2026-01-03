// pkg/urn/urn.go
package urn

import (
	"strings"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/fault"
)

// URN represents a fully parsed and validated Unkey URN
type URN struct {
	Namespace    string       // Always "unkey" for standard resources
	Service      Service      // The service namespace
	WorkspaceID  string       // Workspace identifier
	Environment  string       // Environment (production, staging, etc.)
	ResourceType ResourceType // Type of resource
	ResourceID   string       // Resource identifier
}

// String converts a URN to its string representation
func (u URN) String() string {
	return "urn:" + u.Namespace + ":" + string(u.Service) + ":" +
		u.WorkspaceID + ":" + u.Environment + ":" +
		string(u.ResourceType) + "/" + u.ResourceID
}

// ServiceStr returns the service component of the URN
func (u URN) ServiceStr() string {
	return string(u.Service)
}

// ResourceTypeStr returns the resource type component of the URN
func (u URN) ResourceTypeStr() string {
	return string(u.ResourceType)
}

// ValidateNamespace checks namespace is valid
func validateNamespace(namespace string) error {
	return assert.Equal(namespace, "unkey", "namespace must be 'unkey'")
}

// ValidateWorkspaceID checks workspace ID format
func validateWorkspaceID(id string) error {
	if err := assert.NotEmpty(id, "workspace ID cannot be empty"); err != nil {
		return err
	}
	return assert.True(strings.HasPrefix(id, "ws_"),
		"workspace ID must start with 'ws_' prefix")
}

// ValidateEnvironment checks environment name is valid
func validateEnvironment(env string) error {
	if err := assert.NotEmpty(env, "environment cannot be empty"); err != nil {
		return err
	}
	return nil
}

// ValidateResourceID validates resource ID format based on type
func validateResourceID(resourceType ResourceType, id string) error {
	if err := assert.NotEmpty(id, "resource ID cannot be empty"); err != nil {
		return err
	}

	// Resource type-specific prefix validation
	switch resourceType {
	case ResourceTypeKey:
		return assert.True(strings.HasPrefix(id, "key_"),
			"key ID must start with 'key_' prefix")
	case ResourceTypeAPI:
		return assert.True(strings.HasPrefix(id, "api_"),
			"API ID must start with 'api_' prefix")
	case ResourceTypeNamespace:
		return assert.True(strings.HasPrefix(id, "ns_"),
			"namespace ID must start with 'ns_' prefix")
	case ResourceTypeIdentity:
		return assert.True(strings.HasPrefix(id, "id_"),
			"identity ID must start with 'id_' prefix")
	default:
		// Generic check for other resource types
		return nil
	}
}

// Parse parses a URN string into a structured URN with comprehensive validation
func Parse(urnStr string) (URN, error) {

	// Split the URN into components
	parts := strings.Split(urnStr, ":")

	// Validate basic structure
	if err := assert.Equal(len(parts), 6, "URN must have exactly 6 components separated by ':'"); err != nil {
		return URN{}, fault.Wrap(err, fault.Internal("invalid URN format"), fault.Public(urnStr))
	}

	if err := assert.Equal(parts[0], "urn", "URN must start with 'urn:'"); err != nil {
		return URN{}, fault.Wrap(err, fault.Internal("invalid URN prefix"), fault.Public(urnStr))
	}

	// Extract and validate resource type and ID
	resourcePath := strings.Split(parts[5], "/")
	if err := assert.Equal(len(resourcePath), 2, "resource path must be in format 'type/id'"); err != nil {
		return URN{}, fault.Wrap(err, fault.Internal("invalid resource path format"), fault.Public(parts[5]))
	}

	// Create URN components
	namespace := parts[1]
	service := Service(parts[2])
	workspaceID := parts[3]
	environment := parts[4]
	resourceType := ResourceType(resourcePath[0])
	resourceID := resourcePath[1]

	// Validate all components
	err := assert.All(
		validateNamespace(namespace),
		service.Validate(),
		validateWorkspaceID(workspaceID),
		validateEnvironment(environment),
		resourceType.ValidateForService(service),
		validateResourceID(resourceType, resourceID),
	)

	if err != nil {
		return URN{}, fault.Wrap(err, fault.Internal("invalid URN component"), fault.Public(urnStr))
	}

	// All validations passed, return the URN
	return URN{
		Namespace:    namespace,
		Service:      service,
		WorkspaceID:  workspaceID,
		Environment:  environment,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}, nil
}

// New creates a new validated URN with comprehensive assertions
func New(service Service, workspaceID, environment string, resourceType ResourceType, resourceID string) (URN, error) {

	// Validate all components
	err := assert.All(
		service.Validate(),
		validateWorkspaceID(workspaceID),
		validateEnvironment(environment),
		resourceType.ValidateForService(service),
		validateResourceID(resourceType, resourceID),
	)

	if err != nil {
		return URN{}, fault.Wrap(err, fault.Internal("invalid URN component"))
	}

	// All validations passed
	return URN{
		Namespace:    "unkey",
		Service:      service,
		WorkspaceID:  workspaceID,
		Environment:  environment,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}, nil
}
