# Logdrain

- Commit a nonempty page's cursor only after the sink acknowledges all its
  events. A successful delivery followed by a failed commit can be delivered
  again: the contract is at-least-once. See [engine.go](internal/engine/engine.go).
- Paging uses `(inserted_at, event_id)` with matching ordering in ClickHouse and
  MySQL. Timestamp-only cursors skip equal-timestamp events. Apply workspace and
  stream filters in the source query before limiting, not after reading a page.
- Empty pages advance to the exclusive window end with an empty event ID, so
  events at that boundary remain eligible. See
  [batch_reader.go](internal/engine/batch_reader.go).
- Delivery-state reads and writes are fenced by acquisition token and lease
  expiry using MySQL time. Reacquiring a lease needs a new token even in the same
  process. Stop work when a fenced write affects no rows. Keep the failure-write
  status guard so stale work cannot undo a user's pause. See
  [queries](internal/db/queries/).
- Credential decryption and its cache are scoped to the drain's workspace.
  Generic HTTP sinks use the SSRF-aware transport for customer URLs; validating
  the initial URL alone is not a substitute. The unsafe endpoint option is for
  tests/local development. See [factory.go](internal/engine/factory.go) and
  [httpdrain.go](sink/httpdrain/httpdrain.go).
