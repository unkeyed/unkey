package urn

import (
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ResourceType represents the type of resource within a service
type ResourceType string

const (
	ResourceTypeKey       ResourceType = "key"
	ResourceTypeAPI       ResourceType = "api"
	ResourceTypeNamespace ResourceType = "namespace"
	ResourceTypeIdentity  ResourceType = "identity"
	// Add more as needed
)

// ValidateForService checks that a resource type is valid for a specific service
func (rt ResourceType) ValidateForService(service Service) error {
	// Service-specific resource type validation map
	validResourceTypes := map[Service]map[ResourceType]bool{
		ServiceAuth: {
			ResourceTypeKey: true,
			ResourceTypeAPI: true,
		},
		ServiceRateLimit: {
			ResourceTypeNamespace: true,
		},
		ServiceIdentity: {
			ResourceTypeIdentity: true,
		},
		ServiceDeploy:  {},
		ServiceObserve: {},
		ServiceAudit:   {},
		ServiceSecrets: {},
		ServiceBilling: {},
	}

	// First validate the service itself
	if err := service.Validate(); err != nil {
		return err
	}

	// Then check if this ResourceType is valid for the service
	validTypes, serviceExists := validResourceTypes[service]
	if !serviceExists {
		return fault.New("service has no valid resource types defined",
			fault.Code(codes.App.Validation.AssertionFailed.URN()))
	}

	return assert.True(validTypes[rt],
		"invalid resource type for service "+string(service))
}
