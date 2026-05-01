-- name: BlocklistInsert :exec
-- BlocklistInsert records a single rate-limit denial so other regions can
-- propagate it. The unique key spans (workspace, namespace, identifier,
-- duration_ms, sequence); duplicate inserts return ER_DUP_ENTRY (1062),
-- which the caller may handle as appropriate.
--
-- Production hot path uses [BulkInsertBlocklist] in database.go. This
-- single-row form is kept for tests that need to seed individual rows
-- directly.
INSERT INTO ratelimit_blocklist (
    workspace_id,
    namespace,
    identifier,
    duration_ms,
    sequence,
    `limit`,
    expires_at
) VALUES (
    sqlc.arg("workspace_id"),
    sqlc.arg("namespace"),
    sqlc.arg("identifier"),
    sqlc.arg("duration_ms"),
    sqlc.arg("sequence"),
    sqlc.arg("limit"),
    sqlc.arg("expires_at")
);
