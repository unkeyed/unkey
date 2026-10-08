// Package rootkey persists new root keys for the API.
//
// [Insert] writes the root key row, its permission rows, and the audit logs for
// both inside the caller's transaction, so a key never exists without its
// permissions or its audit trail. The rootKeys.createKey endpoint and the CLI
// device login both create keys through it, which keeps the two paths from
// drifting. Callers authorize the request and validate the permissions before
// calling [Insert].
//
// # Usage
//
//	err := db.TxRetry(ctx, database.RW(), func(ctx context.Context, tx db.DBTX) error {
//		created, err := rootkey.Insert(ctx, tx, rootkey.InsertRequest{ /* ... */ })
//		return err
//	})
package rootkey
