// Package grant implements the device login that `unkey login` uses.
//
// The CLI starts a login and polls with a secret id. A signed-in dashboard
// user compares the code, narrows the permissions to a subset of their role,
// and allows or denies it. The first poll after approval creates a CLI session
// and returns a short-lived access token plus a rotating refresh token. The
// refresh token is stored only as a hash.
//
// Root keys are not minted here. Machines keep using root keys.
package grant
