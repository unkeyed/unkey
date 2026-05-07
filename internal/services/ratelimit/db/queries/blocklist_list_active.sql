-- name: BlocklistListActive :many
-- BlocklistListActive returns every still-active denial for the sync loop to
-- apply locally. The result set is bounded by unique violators currently in
-- their TTL window; the expires_at index keeps the scan cheap.
SELECT
    workspace_id,
    namespace,
    identifier,
    duration_ms,
    sequence,
    `limit`,
    expires_at
FROM ratelimit_blocklist
WHERE expires_at > sqlc.arg("now");
