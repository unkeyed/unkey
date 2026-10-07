// Package cdc copies MySQL rows and follows changes through Vitess VStream.
// Callers choose the tables and filters, then apply or relay the events.
// Events include field types and old row values, including deleted rows.
//
// # Usage
//
// Create one watcher per consumer. Each watcher owns its connection.
// With a sendEvent function already set up, a local Vitess watch looks like:
//
//	watcher, err := cdc.New(cdc.Config{
//		Address: "localhost:33575",
//		Keyspace: "unkey",
//		Insecure: true,
//		Rules: []cdc.Rule{{Table: "records", Query: "select id, value from records"}},
//	})
//	if err != nil {
//		return err
//	}
//	err = watcher.Watch(ctx, nil, sendEvent)
//	return errors.Join(err, watcher.Close())
//
// # Recovery
//
// An empty token starts a copy of matching rows followed by live changes.
// Watch sends FIELD and ROW changes and checkpoint tokens in order. It does not
// save progress or retry automatically. The consumer must save a checkpoint
// only after applying all earlier changes, then supply that token on reconnect.
// A successful relay send does not mean the consumer has applied the change.
//
// Tokens can resume a partial initial copy. Changes can repeat, so consumers
// must handle them safely. After [ErrInvalidToken] or [ErrExpired], retry with
// an empty token and remove destination records that no longer exist.
//
// Overlapping Watch calls return an error instead of waiting. Close the watcher
// when it is no longer needed. Close can interrupt an active stream, but it does
// not wait for a callback already in progress.
package cdc
