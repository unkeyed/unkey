/*
Package codes provides a system for categorizing, identifying, and managing
error codes throughout the Unkey platform. It implements a uniform error code
structure in a URN-like format (err:system:category:specific) that enables
consistent error reporting, documentation, and handling.

Error codes are hierarchically organized by system (e.g., unkey, user, github),
category (e.g., authentication, data), and specific error type. This structure
facilitates both programmatic error handling with type safety and human-readable
error messages that can be looked up in documentation.

This package is used across all Unkey services to ensure consistency in error
reporting and handling. It also provides utilities for parsing error codes from
strings and generating documentation URLs.
*/
package codes
