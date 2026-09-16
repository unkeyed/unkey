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
//		ResumeToken: savedToken,
//	})
//	if err != nil {
//		return err
//	}
//	return client.Watch(ctx, applyChange)
//
// # Recovery
//
// Watch passes only FIELD and ROW changes to the callback. It saves checkpoints
// in memory after earlier callbacks succeed. Finish applying each change before
// returning. Calling Watch again resumes from that position, even during the
// initial copy. The client does not retry automatically.
//
// After Watch stops, [Client.ResumeToken] returns a token you can persist for
// process restarts. Store it as-is. Tokens check that the filter has not changed,
// but do not grant access. Client methods must not run concurrently.
//
// An empty token starts a snapshot: a new copy of the matching rows.
// New returns [ErrInvalidToken] for a rejected token. Watch returns [ErrExpired]
// if its position is no longer available. Both require a new client with an empty
// token. Callers must also remove destination records that no longer exist.
//
// Relays use [Client.Forward] to send both changes and checkpoints downstream.
// Sending an event does not prove it was applied. On a downstream reconnect,
// create a client with that consumer's saved token, not the relay's last token.
// Both Watch and Forward can repeat changes; callers must handle them safely.
package cdc
