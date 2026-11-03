// Package clidevice implements the logins that `unkey login` and MCP clients
// use, and the device sessions they create.
//
// The CLI starts a device login and polls with a secret id. An MCP client
// starts an OAuth 2.1 authorization code login instead, identified by a
// client metadata document URL or a stateless dynamic registration, and
// redeems the code with PKCE. Both land on the same dashboard page, where a
// signed-in user narrows the permissions to a subset of their role and allows
// or denies the login. Redeeming the approval creates a session whose kind,
// cli or mcp, is fixed by the grant, and returns a short-lived access token
// plus a rotating refresh token. The refresh token is stored only as a hash.
//
// Root keys are not minted here. Machines keep using root keys.
package clidevice
