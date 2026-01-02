package otel

import "fmt"

// NewSpanName creates a standardized span name by joining package and method names.
// This ensures consistent span naming across the application, making it easier
// to filter and analyze traces.
//
// The format is "{package}.{method}", e.g., "db.query" or "api.verify_key".
//
// Parameters:
//   - pkg: The package or module name
//   - method: The method or function name
//
// Example:
//
//	// In a database query function
//	ctx, span := tracing.Start(ctx, otel.NewSpanName("db", "findUserByID"))
//	defer span.End()
//
//	// In an API handler
//	ctx, span := tracing.Start(ctx, otel.NewSpanName("api", "verifyKey"))
//	defer span.End()
func NewSpanName(pkg string, method string) string {
	return fmt.Sprintf("%s.%s", pkg, method)
}
