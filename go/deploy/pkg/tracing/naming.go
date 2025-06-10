// Package tracing provides utilities for parsing and formatting distributed tracing span names
// from Connect RPC procedure paths.
//
// This package standardizes span naming conventions across Unkey services by extracting
// service and method information from RPC procedure paths and formatting them into
// consistent span names for observability.
//
// Example usage:
//
//	procedure := "/metald.v1.VmService/CreateVm"
//	method := tracing.ExtractMethodName(procedure)    // "CreateVm"
//	service := tracing.ExtractServiceName(procedure)  // "metald.v1.VmService"
//	span := tracing.FormatSpanName("metald", method)  // "metald.CreateVm"
package tracing

import "strings"

// ExtractMethodName extracts the method name from a Connect RPC procedure path.
// It returns the last path segment after the final slash, or the entire procedure
// string if no slash is found.
//
// Example:
//
//	ExtractMethodName("/metald.v1.VmService/CreateVm") // returns "CreateVm"
//	ExtractMethodName("CreateVm")                      // returns "CreateVm"
func ExtractMethodName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return procedure
}

// ExtractServiceName extracts the service name from a Connect RPC procedure path.
// It returns the first path segment after the leading slash, or an empty string
// if the procedure path has fewer than two segments.
//
// Example:
//
//	ExtractServiceName("/metald.v1.VmService/CreateVm") // returns "metald.v1.VmService"
//	ExtractServiceName("/CreateVm")                    // returns ""
//	ExtractServiceName("invalid")                      // returns ""
func ExtractServiceName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// FormatSpanName creates a standardized span name by combining a service name
// and method name with a dot separator. This provides consistent span naming
// across all Unkey services for distributed tracing.
//
// Example:
//
//	FormatSpanName("metald", "CreateVm")     // returns "metald.CreateVm"
//	FormatSpanName("billaged", "GetUsage")  // returns "billaged.GetUsage"
func FormatSpanName(serviceName, methodName string) string {
	return serviceName + "." + methodName
}
