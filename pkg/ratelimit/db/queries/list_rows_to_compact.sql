-- name: ListRowsToCompact :many
-- ListRowsToCompact pages through rows whose compact_after_ms has passed, in
-- the order of the compact_after_ms index. A page resumes after the last row of
-- the previous page. For the first page, pass an after_compact_after_ms of -1;
-- the after_fingerprint then does not matter. A page can hold several rows of
-- one fingerprint, so callers remove duplicates and then handle the whole
-- fingerprint with ListSessions.
SELECT fingerprint, compact_after_ms
FROM ratelimit_state
WHERE compact_after_ms < sqlc.arg(now_ms)
    AND compact_after_ms >= sqlc.arg(after_compact_after_ms)
    AND (compact_after_ms > sqlc.arg(after_compact_after_ms) OR fingerprint > sqlc.arg(after_fingerprint))
ORDER BY compact_after_ms, fingerprint
LIMIT ?;
