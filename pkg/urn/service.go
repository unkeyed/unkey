package urn

import (
	"github.com/unkeyed/unkey/pkg/assert"
)

// Service represents a valid Unkey service namespace
type Service string

const (
	ServiceAuth      Service = "auth"
	ServiceRateLimit Service = "ratelimit"
	ServiceIdentity  Service = "identity"
	ServiceDeploy    Service = "deploy"
	ServiceObserve   Service = "observe"
	ServiceAudit     Service = "audit"
	ServiceSecrets   Service = "secrets"
	ServiceBilling   Service = "billing"
)

// All valid services for quick lookup
var validServices = map[Service]bool{
	ServiceAuth:      true,
	ServiceRateLimit: true,
	ServiceIdentity:  true,
	ServiceDeploy:    true,
	ServiceObserve:   true,
	ServiceAudit:     true,
	ServiceSecrets:   true,
	ServiceBilling:   true,
}

// Validate ensures the service is valid
func (s Service) Validate() error {
	return assert.True(validServices[s],
		"invalid service: must be one of [auth, ratelimit, identity, deploy, observe, audit, secrets, billing]")
}
