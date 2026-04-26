// Package httpx is a thin generic JSON HTTP client. It exists to
// kill the ten-line boilerplate that surrounds every JSON HTTP call
// in this codebase: build *http.Request, set Content-Type +
// Accept + Authorization, Do, defer Body.Close, check status,
// io.ReadAll, json.Unmarshal.
//
// Scope is deliberately narrow:
//
//   - JSON in, JSON out (or no body either way).
//   - Status code branching via a typed StatusError.
//   - Per-call options for headers and request mutation.
//
// Out of scope: retries (compose with pkg/retry), circuit breaking,
// metrics, tracing, multipart, streaming. Add those at the call
// site, not here.
//
// The helpers are top-level generic functions rather than methods on
// a Client, because Go forbids type parameters on methods. For
// repeat callers (e.g. a GitHub App client), construct your own
// helper that prepends a base URL and default headers, and forwards
// to httpx. See svc/preflight/internal/githubpush for an example.
//
// # Errors
//
// Any non-2xx response yields a *StatusError. Inspect with errors.As
// or the IsStatus shortcut:
//
//	user, err := httpx.Get[User](ctx, "https://api.example.com/users/me")
//	if httpx.IsStatus(err, http.StatusNotFound) {
//	    // handle missing user
//	}
//	if err != nil {
//	    return err
//	}
//
// Network errors and JSON decode errors are returned verbatim and
// are NOT *StatusError; callers that want to distinguish "server
// said no" from "couldn't reach server" check for *StatusError
// explicitly.
package httpx
