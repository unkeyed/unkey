-- name: BlocklistInsert :exec
-- BlocklistInsert records a single rate-limit denial so other regions can
-- propagate it. The unique key spans (workspace, namespace, identifier,
-- duration_ms, sequence), so cross-region concurrent emits at the same
-- sequence collide and are silently dropped — INSERT IGNORE is sufficient
-- because every emit for the same key carries identical content (limit and
-- expires_at are both deterministic at the application layer).
--
-- Sequence advancement (sustained abuse crossing window boundaries) creates
-- a new row instead of mutating an existing one. Receivers sync all active
-- rows and inflate the matching counter for each.
INSERT IGNORE INTO ratelimit_blocklist (
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
