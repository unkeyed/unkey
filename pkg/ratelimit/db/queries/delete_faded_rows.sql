-- name: DeleteFadedRows :execrows
-- DeleteFadedRows deletes at most LIMIT rows of any fingerprint whose level
-- has faded, by the MySQL clock that the caller read. It uses the
-- delete_after_ms index, so a pass stays bounded however large the table is.
DELETE FROM ratelimit_state
WHERE delete_after_ms < sqlc.arg(before_ms)
LIMIT ?;
