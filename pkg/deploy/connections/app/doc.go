// Package app defines naming rules for app-to-app connections and their
// generated hostname variables. Ctrl uses these rules when creating deployment
// snapshots; deployment target selection belongs to Ctrl, not this package.
//
// # Usage
//
// Validate names with [IsValidName] before generating a variable:
//
//	key, host := app.HostVariable("worker")
//	// key is WORKER_HOST; host is worker.unkey.internal.
package app
