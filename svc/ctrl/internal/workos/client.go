package workos

import (
	"net/http"
	"time"
)

// baseURL is the WorkOS REST API origin.
const baseURL = "https://api.workos.com"

// adminRoleSlug is the membership role that receives billing alerts; matches the
// dashboard's own admin check.
const adminRoleSlug = "admin"

// membershipPageSize bounds each org-memberships page.
const membershipPageSize = 100

// requestTimeout bounds each WorkOS call. The lookup runs inside a journaled
// Restate step on a budget-threshold crossing; a hung request should fail the
// step and retry rather than stall the invocation.
const requestTimeout = 30 * time.Second

// client is the WorkOS-backed Resolver.
type client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

var _ Resolver = (*client)(nil)

// New returns a WorkOS-backed Resolver. apiKey must be non-empty; a caller
// with no key configured wires NewNoop instead and owns that decision.
func New(apiKey string) Resolver {
	return &client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: requestTimeout}, //nolint:exhaustruct // default transport, only the timeout matters
	}
}
