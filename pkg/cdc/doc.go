// Package cdc copies MySQL rows and follows changes through Vitess VStream.
// Callers choose the tables and filters, then apply the events they receive.
// Events include field types and old row values, including deleted rows.
//
// # Usage
//
// Each [Client.Watch] opens a VStream on the client's shared connection.
// With a client and callbacks already set up:
//
//	rules := []cdc.Rule{{Table: "records", Query: "select id, value from records"}}
//	err := client.Watch(ctx, rules, resumeToken, handleEvent, saveCheckpoint)
//
// # Recovery
//
// Apply events in order. Save a checkpoint only after all earlier changes have
// been saved at the destination. Reconnect with that token to continue, even
// during the initial copy. Store tokens as-is. They check that the filter has
// not changed, but do not grant access.
//
// An empty token starts a snapshot: a new copy of the matching rows.
// [ErrInvalidToken] and [ErrExpired] require a new snapshot. Callers must also
// remove destination records that no longer exist in MySQL.
//
// This package does not retry or store events. Callers must reconnect and
// handle repeated events safely.
package cdc
