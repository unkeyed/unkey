// Package grant runs delegated user grants. A signed-in Unkey user gives a
// holder a subset of their permissions in one workspace. The holder gets a
// session: a 15-minute access token the whole API accepts and a rotating
// refresh token stored only as a hash.
//
// A grant starts as a request and reaches its holder one of two ways:
//
//   - Device. The Unkey CLI starts the request (cli.startDeviceLogin), shows a
//     user code, and polls with the request's secret id (cli.exchangeToken).
//   - Redirect. A registered OAuth client sends the user's browser to the
//     authorization endpoint. Approval mints a single-use authorization code
//     and sends the browser back to the client's redirect URI with it. The
//     client redeems the code with its PKCE verifier at the token endpoint.
//
// Both modes share dashboard approval (narrowed to the approver's role),
// issuance (WorkOS role re-check, session row, audit), refresh rotation, the
// revocation cache, and the sessions list.
//
// Storage keeps the device login names because PlanetScale cannot rename:
// requests live in device_authorizations, sessions in cli_sessions, and
// client_id is NULL for the Unkey CLI in both.
//
// Root keys are not minted here. Machines keep using root keys.
package grant
