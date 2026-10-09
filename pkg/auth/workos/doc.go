// Package workos maps WorkOS organization roles to API permissions after JWT
// verification. The API enables this mapping through the JWT provider config.
// A permission ceiling on that config narrows the mapped permissions and
// never adds permissions the role does not already grant.
package workos
