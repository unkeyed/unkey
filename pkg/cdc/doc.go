// Package cdc copies MySQL rows and follows changes through Vitess VStream.
// Callers choose the tables and filters, then apply the events they receive.
// Events include field types and old row values, including deleted rows.
//
// # Usage
//
// Create one client per consumer. Each client owns its connection.
// With an applyChange function already set up, a local Vitess watch looks like:
//
//	client, err := cdc.New(cdc.Config{
//		Address: "localhost:33575",
//		Keyspace: "unkey",
//		Insecure: true,
//		Rules: []cdc.Rule{{Table: "records", Query: "select id, value from records"}},
//	})
//	if err != nil {
//		return err
//	}
//	err = client.Watch(ctx, applyChange)
//	return errors.Join(err, client.Close())
//
// # Recovery
//
// The first Watch copies matching rows, then follows live changes. It passes
// only FIELD and ROW changes to the callback. Finish applying each change before
// returning. The client saves checkpoints in memory after callbacks succeed.
//
// Reuse the same client for retries. It resumes from its last checkpoint, even
// during the initial copy. Watch does not expose tokens or retry automatically.
// Calls on the same client must not overlap. Close it after all retries finish.
//
// After [ErrExpired], the next Watch starts a fresh snapshot on the same client.
// The caller must also remove destination records that no longer exist.
//
// Relays use [Client.Forward] with the downstream consumer's resume token.
// It sends both changes and checkpoints without saving progress in the relay.
// Both Watch and Forward can repeat changes; callers must handle them safely.
package cdc
