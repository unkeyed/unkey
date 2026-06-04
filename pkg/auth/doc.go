// Package auth normalizes API credentials into principals.
//
// A [Service] tries configured [Resolver] implementations in order. This lets
// API handlers authenticate root keys, portal sessions, and scaffolded
// credential sources through one request flow. Principals are defined by
// [github.com/unkeyed/unkey/pkg/auth/principal.Principal] so request sessions
// and auth services can share the type without creating an import cycle.
package auth
