// Package clidevice implements the device login that `unkey login` uses to
// obtain a root key without the user pasting one into a terminal.
//
// The CLI calls [Service.Start] and shows the user code. A signed-in dashboard
// user confirms the code and picks permissions, which calls [Service.Approve].
// The CLI polls [Service.Poll] with its login ID, and the first poll after
// approval mints the root key and returns it exactly once. Only a hash of the
// key is stored; the login row keeps the key ID so an undelivered key can be
// found and deleted.
//
// # Security model
//
// The login ID is the CLI's bearer secret and never leaves the CLI. The user
// code is short-lived and only identifies the login to the dashboard.
// [Service.Get], [Service.Approve], and [Service.Deny] require a dashboard
// session, so a root key can never approve or inspect a login. Approval needs
// root key write permission, and every granted permission must fit inside the
// approver's own. Start and Poll are unauthenticated, so both are rate limited
// per IP address. The approval page shows the requester's IP address and user
// agent to help users spot a login they did not start.
//
// # Lifecycle
//
// A login moves from pending to approved to consumed, or to denied. Expiry is
// derived from expires_at rather than stored. Start deletes logins that expired
// more than a day ago, in small batches.
//
// # Usage
//
//	started, err := svc.Start(ctx, clidevice.StartRequest{DeviceName: host, RemoteIP: ip, UserAgent: ua})
//	// show started.UserCode and started.VerificationURIComplete
//	result, err := svc.Poll(ctx, started.LoginID, ip, ua)
//	if result.Status == clidevice.PollComplete {
//		// store *result.Key
//	}
package clidevice
