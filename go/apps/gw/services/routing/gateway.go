package routing

import (
	"context"
	"net/url"
)

// TargetInfo represents the gateway configuration and available VMs.
type TargetInfo struct {
	// GatewayID is the unique identifier for this gateway
	GatewayID string

	// WorkspaceID identifies which workspace this gateway belongs to
	WorkspaceID string

	// Name is a human-readable name for the gateway
	Name string

	// Enabled indicates whether this gateway is active
	Enabled bool

	// AvailableVMs are the backend VMs that can handle requests
	AvailableVMs []string

	// Metadata contains additional gateway configuration
	Metadata map[string]string
}


// Service handles gateway configuration lookup and VM selection.
type Service interface {
	// GetTargetByHost finds gateway configuration based on the request host
	GetTargetByHost(ctx context.Context, host string) (*TargetInfo, error)

	// SelectVM picks an available VM from the gateway's VM list
	SelectVM(ctx context.Context, targetInfo *TargetInfo) (*url.URL, error)
}

// RequestContext provides information about an incoming request for routing decisions.
type RequestContext struct {
	Host    string
	Path    string
	Method  string
	Headers map[string]string
}
