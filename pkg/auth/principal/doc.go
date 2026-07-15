// Package principal defines the normalized authenticated subject shared by API
// authentication, request sessions, and audit logging.
//
// The package is intentionally independent from pkg/auth and pkg/zen so both
// layers can refer to the same principal type without creating an import cycle.
// Authentication resolvers construct a [Principal], auth services authorize it,
// and request sessions retain it as request-wide metadata for logging and
// compatibility helpers.
package principal
