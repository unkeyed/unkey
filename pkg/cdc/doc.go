// Package cdc copies MySQL rows and follows changes through Vitess VStream.
// Callers choose the tables and filters, then apply the events they receive.
// Events include field types and old row values, including deleted rows.
//
// # Usage
//
// Create a [Connection] with [NewConnection], then configure one client per
// consumer. With a connection and an applyChange function already set up:
//
//	client, err := cdc.New(cdc.Config{
//		Connection: connection,
//		Rules: []cdc.Rule{{Table: "records", Query: "select id, value from records"}},
//	})
//	if err != nil {
//		return err
//	}
//	return client.Watch(ctx, applyChange)
//
// # Recovery
//
// The first Watch copies matching rows, then follows live changes. It passes
// only FIELD and ROW changes to the callback. Finish applying each change before
// returning. The client saves checkpoints in memory after callbacks succeed.
//
// Reuse the same client for retries. It resumes from its last checkpoint, even
// during the initial copy. There is no token import, export, or automatic retry.
// Watch calls on the same client must not overlap.
//
// After [ErrExpired], the next Watch starts a fresh snapshot on the same client.
// The caller must also remove destination records that no longer exist.
//
// Relays use [Connection.Forward] with the downstream consumer's resume token.
// It sends both changes and checkpoints without saving progress in the relay.
// Both Watch and Forward can repeat changes; callers must handle them safely.
package cdc
