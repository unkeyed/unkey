// Package tracing provides unified tracing utilities for Unkey services
package tracing

import "strings"

// ExtractMethodName extracts the method name from a full procedure path
// e.g., "/vmprovisioner.v1.VmService/CreateVm" -> "CreateVm"
func ExtractMethodName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return procedure
}

// ExtractServiceName extracts the service name from a full procedure path
// e.g., "/vmprovisioner.v1.VmService/CreateVm" -> "vmprovisioner.v1.VmService"
func ExtractServiceName(procedure string) string {
	parts := strings.Split(procedure, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// FormatSpanName creates a unified span name for RPC operations
// e.g., "metald", "CreateVm" -> "metald.CreateVm"
func FormatSpanName(serviceName, methodName string) string {
	return serviceName + "." + methodName
}
