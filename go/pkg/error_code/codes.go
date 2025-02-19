// Package errorcode provides error classification
package errorcode

import (
	"encoding/json"
	"fmt"
)

// System represents the source system of an error
type System string

// Supported system constants
const (
	SystemUnkey  System = "UNKEY"  // Unkey system errors
	SystemAWS    System = "AWS"    // AWS-related errors
	SystemGitHub System = "GITHUB" // GitHub-related errors
)

// Namespace represents the functional area where an error occurred
type Namespace string

// Supported namespace constants
const (
	// Unkey system namespaces
	NamespaceDeployment Namespace = "DEPLOYMENT" // Deployment-related errors
	NamespaceDatabase   Namespace = "DATABASE"   // Database-related errors
	NamespaceKey        Namespace = "KEY"        // Key-related errors

	// AWS namespaces (to be defined)

	// GitHub namespaces (to be defined)
)

// Subsystem represents a specific component within a namespace
type Subsystem string

// base defines the core structure of an error
type base struct {
	// System identifies the source system where the error originated
	System System `json:"system"`

	// Namespace identifies the functional area where the error occurred
	Namespace Namespace `json:"namespace"`

	// Name is the specific identifier for this error
	Name string `json:"name"`

	// Description provides a human-readable explanation of the error
	Description string `json:"description"`

	// PublicMeta contains additional public information about the error
	// that can be safely exposed to end users
	PublicMeta map[string]any `json:"publicMeta"`

	// InternalMeta contains sensitive internal information about the error
	// that should not be exposed to end users
	InternalMeta map[string]any `json:"internalMeta"`

	// Code is a formatted string combining System:Namespace:Name
	// for easy error identification
	Code string `json:"code"`

	// Cause is the original error that caused this.
	Cause error `json:"cause"`
}

// newBase creates a new base error with the given parameters
func newBase(
	err error,
	system System,
	namespace Namespace,
	name string,
	description string,
) base {
	return base{
		System:       system,
		Namespace:    namespace,
		Name:         name,
		Description:  description,
		PublicMeta:   map[string]any{},
		InternalMeta: map[string]any{},
		Code:         fmt.Sprintf("EID:%s:%s:%s", system, namespace, name),
		Cause:        err,
	}

}

// Marshall converts the base error to JSON
func (e base) Marshall() ([]byte, error) {
	return json.Marshal(e)

}
