// Package portal provides session validation for the customer portal.
//
// Portal sessions are browser-based authentication tokens that scope API
// requests to a specific workspace and external user identity. They are
// created via the portal.createSession endpoint and exchanged for
// long-lived browser sessions via portal.exchangeSession.
//
// This service handles session lookup, caching (SWR), and permission/metadata
// deserialization. It is intentionally separate from the keys service because
// portal sessions are not API keys — they have different lifecycles, validation
// rules, and domain semantics.
package portal
