// Package workos maps WorkOS organization roles to API permissions after JWT
// verification. The API enables this mapping through the JWT provider config.
//
// The "workos" provider reads roles from dashboard session tokens. The
// "workos_agent" provider accepts AuthKit agent access tokens and acts for the
// user who claimed the agent registration, granting that user's current roles
// from the WorkOS API rather than any role or scope carried by the token.
package workos
