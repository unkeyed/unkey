// Package uid provides utilities for generating unique identifiers with
// various prefixes for different resource types. These identifiers are used
// throughout the system to uniquely identify resources like API keys, workspaces,
// and requests.
//
// The implementation uses KSUID (K-Sortable Unique Identifiers) as the base
// for all generated IDs. KSUIDs are globally unique, time-sortable identifiers
// that are URL-safe and provide a good balance between uniqueness, readability,
// and size.
//
// Identifiers are typically prefixed with a resource type indicator (e.g., "req_"
// for request IDs, "ins_" for node IDs), making them easily recognizable and
// categorizable. This helps with debugging and logging by making it immediately
// clear what type of resource an ID refers to.
//
// Example usage:
//
//	// Generate a request ID
//	requestID := uid.Request()     // returns "req_1z4UVH4AQfoDtVnFZ9VERXeYGSY"
//
//
//	// Generate a custom ID with a specific prefix
//	customID := uid.New("custom") // returns "custom_1z4UVH4C7Bgt8NsssqZxTTVIiWf"
package uid
