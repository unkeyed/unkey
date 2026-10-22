// Package rootkey persists new root keys for the API.
//
// [Insert] opens its own transaction and writes the root key row, its
// permission rows, and the audit logs for both, so a key never exists without
// its permissions or its audit trail. Callers authorize the request and
// validate the permissions before calling [Insert].
package rootkey
