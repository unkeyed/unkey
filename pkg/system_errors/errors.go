package systemerrors

import "fmt"

// Fault identifies the external party or system responsible for an error.
// Use Fault to categorize errors by their origin, enabling consistent error
// attribution and monitoring across integrations.
type Fault string

const (
	// AWS indicates the error originated from Amazon Web Services.
	AWS Fault = "AWS"
	// Unkey indicates the error originated from Unkey's own systems.
	Unkey Fault = "Unkey"
	// GitHub indicates the error originated from GitHub.
	GitHub Fault = "GitHub"
)

// Service identifies the specific service or component where an error occurred.
// Combined with [Fault] and [Code], it enables precise error identification.
type Service string

const (
	// AppRunner identifies AWS App Runner as the error source.
	AppRunner Service = "AppRunner"
	// Route53 identifies AWS Route53 as the error source.
	Route53 Service = "Route53"
	// UnkeyDeploy identifies Unkey's deployment service as the error source.
	UnkeyDeploy Service = "UnkeyDeploy"
)

// Code represents a specific error condition within a service.
// Codes are service-agnostic and can be reused across different services.
type Code string

const (
	// ACCESS_DENIED indicates the operation failed due to insufficient permissions.
	ACCESS_DENIED Code = "ACCESS_DENIED"
)

// EID (Error ID) is a globally unique identifier for errors.
//
// Error IDs follow the format "EID:{Fault}:{Service}:{Code}", for example
// "EID:AWS:Route53:ACCESS_DENIED". Use [Error.EID] to generate an EID from
// an [Error] struct.
type EID string

// Error represents a structured system error with fault attribution.
// It combines [Fault], [Service], and [Code] to uniquely identify error
// conditions across the system. Use the [Error.EID] method to generate
// a string identifier suitable for logging and error tracking.
type Error struct {
	Fault   Fault
	Service Service
	Code    Code
}

// EID returns the globally unique error identifier for this error.
// The returned [EID] follows the format "EID:{Fault}:{Service}:{Code}".
func (e Error) EID() EID {
	return EID(fmt.Sprintf("EID:%s:%s:%s", e.Fault, e.Service, e.Code))
}
