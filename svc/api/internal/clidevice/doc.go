// Package clidevice implements the device login that `unkey login` uses, and
// an OAuth 2.1 authorization server built on the same records.
//
// The CLI starts a login and polls with a secret id. A signed-in dashboard
// user compares the code, narrows the permissions to a subset of their role,
// and allows or denies it. The first poll after approval creates a CLI session
// and returns a short-lived access token plus a rotating refresh token. The
// refresh token is stored only as a hash.
//
// An OAuth client, such as an MCP gateway, starts the same kind of login at
// [Service.Authorize] instead. The client is a public client identified by
// the URL of its Client ID Metadata Document. The dashboard user approves it
// on the same page. Approval mints a single-use authorization code that lives
// 60 seconds, stores only its hash, and redirects the browser to the client
// with it. [Service.ExchangeCode] finds the login by that hash, checks PKCE,
// and creates a device session bound to that client. The device poll treats
// an OAuth login as unknown, so a login is never redeemed without PKCE.
//
// Root keys are not minted here. Machines keep using root keys.
package clidevice
